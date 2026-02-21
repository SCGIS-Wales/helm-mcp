package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ssddgreg/helm-mcp/internal/server"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	mode := flag.String("mode", "stdio", "Transport mode: stdio, http, or sse")
	addr := flag.String("addr", ":8080", "Listen address for http/sse mode")
	showVersion := flag.Bool("version", false, "Print version and exit")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	// Configure logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	if !*debug {
		log.SetOutput(io.Discard)
	} else {
		log.SetOutput(os.Stderr)
		log.Printf("debug logging enabled (version=%s, mode=%s)", version, *mode)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("received signal %v, shutting down", sig)
		cancel()
	}()

	s := server.NewServer()

	switch *mode {
	case "stdio":
		log.Printf("starting stdio server")
		if err := s.Run(ctx, &mcp.StdioTransport{}); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}

	case "http":
		handler := mcp.NewStreamableHTTPHandler(
			func(r *http.Request) *mcp.Server { return s },
			nil,
		)
		httpServer := newHTTPServer(*addr, handler)
		fmt.Fprintf(os.Stderr, "helm-mcp HTTP server listening on %s\n", *addr)
		log.Printf("starting HTTP server on %s", *addr)
		gracefulShutdown(ctx, httpServer)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}

	case "sse":
		handler := mcp.NewSSEHandler(
			func(r *http.Request) *mcp.Server { return s },
			nil,
		)
		httpServer := newHTTPServer(*addr, handler)
		fmt.Fprintf(os.Stderr, "helm-mcp SSE server listening on %s\n", *addr)
		log.Printf("starting SSE server on %s", *addr)
		gracefulShutdown(ctx, httpServer)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "SSE server error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s (valid: stdio, http, sse)\n", *mode)
		os.Exit(1)
	}
}

// gracefulShutdown starts a goroutine that waits for ctx cancellation
// and then shuts down the HTTP server with a 5-second deadline.
func gracefulShutdown(ctx context.Context, srv *http.Server) {
	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}()
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}
}

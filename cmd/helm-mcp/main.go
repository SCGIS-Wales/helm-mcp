package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ssddgreg/helm-mcp/internal/server"
)

func main() {
	mode := flag.String("mode", "stdio", "Transport mode: stdio, http, or sse")
	addr := flag.String("addr", ":8080", "Listen address for http/sse mode")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	s := server.NewServer()

	switch *mode {
	case "stdio":
		if err := s.Run(ctx, &mcp.StdioTransport{}); err != nil {
			log.Fatalf("server error: %v", err)
		}

	case "http":
		handler := mcp.NewStreamableHTTPHandler(
			func(r *http.Request) *mcp.Server { return s },
			nil,
		)
		httpServer := newHTTPServer(*addr, handler)
		fmt.Fprintf(os.Stderr, "helm-mcp HTTP server listening on %s\n", *addr)
		go func() {
			<-ctx.Done()
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			httpServer.Shutdown(shutdownCtx)
		}()
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}

	case "sse":
		handler := mcp.NewSSEHandler(
			func(r *http.Request) *mcp.Server { return s },
			nil,
		)
		httpServer := newHTTPServer(*addr, handler)
		fmt.Fprintf(os.Stderr, "helm-mcp SSE server listening on %s\n", *addr)
		go func() {
			<-ctx.Done()
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			httpServer.Shutdown(shutdownCtx)
		}()
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("SSE server error: %v", err)
		}

	default:
		log.Fatalf("unknown mode: %s (valid: stdio, http, sse)", *mode)
	}
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

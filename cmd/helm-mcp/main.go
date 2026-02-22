package main

import (
	"context"
	"crypto/subtle"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ssddgreg/helm-mcp/internal/security"
	"github.com/ssddgreg/helm-mcp/internal/server"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	mode := flag.String("mode", "stdio", "Transport mode: stdio, http, or sse")
	addr := flag.String("addr", ":8080", "Listen address for http/sse mode")
	showVersion := flag.Bool("version", false, "Print version and exit")
	debug := flag.Bool("debug", false, "Enable debug logging")
	noHarden := flag.Bool("no-harden", false, "Disable process security hardening (for debugging)")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	// Configure structured logging via slog.
	var logHandler slog.Handler
	if *debug {
		logHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		logHandler = slog.NewTextHandler(io.Discard, nil)
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	if *debug {
		slog.Debug("debug logging enabled", "version", version, "mode", *mode)
	}

	// Apply process security hardening (Linux: PR_SET_DUMPABLE, capability dropping).
	// This must happen early, before any sensitive data is loaded.
	hardenResult := security.ApplyHardening(security.HardenOptions{
		DisableAll: *noHarden,
		Debug:      *debug,
	})
	if *debug {
		slog.Debug("security hardening", "result", hardenResult.String())
		for _, e := range hardenResult.Errors {
			slog.Warn("security hardening error", "error", e)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Read optional bearer token from env for HTTP/SSE authentication.
	authToken := os.Getenv("HELM_MCP_AUTH_TOKEN")

	switch *mode {
	case "stdio":
		s := server.NewServer(version)
		slog.Debug("starting stdio server")
		if err := s.Run(ctx, &mcp.StdioTransport{}); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}

	case "http":
		// Each HTTP request gets its own mcp.Server to avoid shared-state
		// concurrency issues across concurrent sessions.
		handler := mcp.NewStreamableHTTPHandler(
			func(r *http.Request) *mcp.Server { return server.NewServer(version) },
			nil,
		)
		httpServer := newHTTPServer(*addr, withAuth(handler, authToken))
		fmt.Fprintf(os.Stderr, "helm-mcp HTTP server listening on %s\n", *addr)
		if authToken != "" {
			fmt.Fprintf(os.Stderr, "  authentication: bearer token (HELM_MCP_AUTH_TOKEN)\n")
		} else {
			fmt.Fprintf(os.Stderr, "  authentication: NONE (set HELM_MCP_AUTH_TOKEN to enable)\n")
		}
		slog.Info("starting HTTP server", "addr", *addr, "auth", authToken != "") //nolint:gosec // addr comes from a trusted CLI flag, not user input
		gracefulShutdown(ctx, httpServer)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}

	case "sse":
		// Each SSE session gets its own mcp.Server.
		handler := mcp.NewSSEHandler(
			func(r *http.Request) *mcp.Server { return server.NewServer(version) },
			nil,
		)
		httpServer := newHTTPServer(*addr, withAuth(handler, authToken))
		fmt.Fprintf(os.Stderr, "helm-mcp SSE server listening on %s\n", *addr)
		if authToken != "" {
			fmt.Fprintf(os.Stderr, "  authentication: bearer token (HELM_MCP_AUTH_TOKEN)\n")
		} else {
			fmt.Fprintf(os.Stderr, "  authentication: NONE (set HELM_MCP_AUTH_TOKEN to enable)\n")
		}
		slog.Info("starting SSE server", "addr", *addr, "auth", authToken != "") //nolint:gosec // addr comes from a trusted CLI flag, not user input
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

// withAuth wraps a handler with bearer token authentication when token is
// non-empty. When no token is configured the handler is returned as-is.
func withAuth(next http.Handler, token string) http.Handler {
	if token == "" {
		return next
	}
	expected := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(auth, expected) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// gracefulShutdown starts a goroutine that waits for ctx cancellation
// and then shuts down the HTTP server with a 5-second deadline.
func gracefulShutdown(ctx context.Context, srv *http.Server) {
	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP server shutdown error", "error", err)
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

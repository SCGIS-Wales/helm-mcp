package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ssddgreg/helm-mcp/internal/resilience"
	"github.com/ssddgreg/helm-mcp/internal/security"
	"github.com/ssddgreg/helm-mcp/internal/server"
	"github.com/ssddgreg/helm-mcp/internal/tools"
)

// version is set at build time via -ldflags "-X main.version=..."
var version = "dev"

func main() {
	mode := flag.String("mode", "stdio", "Transport mode: stdio, http, or sse")
	addr := flag.String("addr", ":8080", "Listen address for http/sse mode")
	showVersion := flag.Bool("version", false, "Print version and exit")
	debug := flag.Bool("debug", false, "Enable debug logging")
	noHarden := flag.Bool("no-harden", false, "Disable process security hardening (for debugging)")
	maxResponseBytes := flag.Int("max-response-bytes", resilience.DefaultMaxResponseBytes,
		"Maximum response size in bytes before truncation (0 to disable)")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	// Configure max response size: CLI flag takes precedence,
	// then HELM_MCP_MAX_RESPONSE_BYTES env var, then the compiled default.
	if envVal := os.Getenv("HELM_MCP_MAX_RESPONSE_BYTES"); envVal != "" && *maxResponseBytes == resilience.DefaultMaxResponseBytes {
		if n, err := strconv.Atoi(envVal); err == nil {
			tools.MaxResponseBytes = n
		} else {
			fmt.Fprintf(os.Stderr, "warning: ignoring invalid HELM_MCP_MAX_RESPONSE_BYTES=%q\n", envVal)
		}
	} else {
		tools.MaxResponseBytes = *maxResponseBytes
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

	// Build the authentication middleware from environment variables.
	// Priority: OIDC > static bearer token > none.
	// Stdio mode is unaffected — auth middleware only applies to HTTP/SSE.
	authMiddleware, sessionCache := buildAuthMiddleware(logger)
	if sessionCache != nil {
		defer sessionCache.Stop()
	}

	switch *mode {
	case "stdio":
		// Stdio mode: no HTTP auth, no breaking changes.
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
		httpServer := newHTTPServer(*addr, authMiddleware(handler))
		printAuthStatus(*addr, "HTTP")
		slog.Info("starting HTTP server", "addr", *addr) //nolint:gosec // addr comes from a trusted CLI flag, not user input
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
		httpServer := newHTTPServer(*addr, authMiddleware(handler))
		printAuthStatus(*addr, "SSE")
		slog.Info("starting SSE server", "addr", *addr) //nolint:gosec // addr comes from a trusted CLI flag, not user input
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

// buildAuthMiddleware constructs the auth middleware from environment variables.
// Returns the middleware function and an optional session cache (caller must Stop() on shutdown).
//
// Environment variables:
//
//	HELM_MCP_OIDC_ISSUER     - OIDC issuer URL (enables OIDC/OAuth2 mode)
//	HELM_MCP_OIDC_AUDIENCE   - Expected audience claim (required with OIDC)
//	HELM_MCP_OIDC_JWKS_URL   - JWKS URL (optional, auto-discovered from issuer)
//	HELM_MCP_REQUIRED_SCOPES - Comma-separated required OAuth2 scopes
//	HELM_MCP_REQUIRED_ROLES  - Comma-separated required app roles
//	HELM_MCP_ALLOWED_CLIENTS - Comma-separated allowed client app IDs (azp)
//	HELM_MCP_AUTH_TOKEN       - Static bearer token (legacy, lower priority than OIDC)
func buildAuthMiddleware(logger *slog.Logger) (func(http.Handler) http.Handler, *security.SessionCache) {
	config := security.AuthMiddlewareConfig{
		AuditLogger: security.NewAuditLogger(logger),
	}

	// Check for OIDC configuration.
	oidcIssuer := os.Getenv("HELM_MCP_OIDC_ISSUER")
	oidcAudience := os.Getenv("HELM_MCP_OIDC_AUDIENCE")

	// Fail fast if OIDC is partially configured — silent auth bypass is a security risk.
	if (oidcIssuer != "") != (oidcAudience != "") {
		fmt.Fprintf(os.Stderr, "fatal: HELM_MCP_OIDC_ISSUER and HELM_MCP_OIDC_AUDIENCE must both be set (or both unset)\n")
		os.Exit(1)
	}

	if oidcIssuer != "" && oidcAudience != "" {
		oidcConfig := security.OIDCConfig{
			IssuerURL: oidcIssuer,
			Audience:  oidcAudience,
			JWKSURL:   os.Getenv("HELM_MCP_OIDC_JWKS_URL"),
		}

		if scopes := os.Getenv("HELM_MCP_REQUIRED_SCOPES"); scopes != "" {
			oidcConfig.RequiredScopes = splitCSV(scopes)
		}
		if roles := os.Getenv("HELM_MCP_REQUIRED_ROLES"); roles != "" {
			oidcConfig.RequiredRoles = splitCSV(roles)
		}
		if clients := os.Getenv("HELM_MCP_ALLOWED_CLIENTS"); clients != "" {
			oidcConfig.AllowedClientIDs = splitCSV(clients)
		}

		validator, err := security.NewOIDCValidator(oidcConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fatal: invalid OIDC configuration: %v\n", err)
			os.Exit(1)
		}

		config.OIDCValidator = validator

		// Enable session cache for OIDC mode.
		sessionCfg := security.DefaultSessionConfig()
		if ttlStr := os.Getenv("HELM_MCP_SESSION_TTL"); ttlStr != "" {
			if d, err := time.ParseDuration(ttlStr); err == nil && d > 0 {
				sessionCfg.InactivityTTL = d
			}
		}
		sessionCache := security.NewSessionCache(sessionCfg)
		config.SessionCache = sessionCache

		return security.NewAuthMiddleware(config), sessionCache
	}

	// Fall back to static bearer token (legacy HELM_MCP_AUTH_TOKEN).
	if token := os.Getenv("HELM_MCP_AUTH_TOKEN"); token != "" {
		config.StaticToken = token
		return security.NewAuthMiddleware(config), nil
	}

	// No auth configured.
	return security.NewAuthMiddleware(config), nil
}

// printAuthStatus logs the authentication mode to stderr.
func printAuthStatus(addr, transport string) {
	fmt.Fprintf(os.Stderr, "helm-mcp %s server listening on %s\n", transport, addr)

	if os.Getenv("HELM_MCP_OIDC_ISSUER") != "" {
		fmt.Fprintf(os.Stderr, "  authentication: OIDC/OAuth2 (issuer=%s)\n", os.Getenv("HELM_MCP_OIDC_ISSUER"))
	} else if os.Getenv("HELM_MCP_AUTH_TOKEN") != "" {
		fmt.Fprintf(os.Stderr, "  authentication: bearer token (HELM_MCP_AUTH_TOKEN)\n")
	} else {
		fmt.Fprintf(os.Stderr, "  authentication: NONE (set HELM_MCP_OIDC_ISSUER or HELM_MCP_AUTH_TOKEN to enable)\n")
	}
}

// splitCSV splits a comma-separated string and trims whitespace from each element.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
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

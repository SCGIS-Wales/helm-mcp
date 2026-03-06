package security

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
)

// contextKey is the type for context keys used by the auth middleware.
type contextKey string

const (
	// ClaimsContextKey is the context key for storing validated token claims.
	ClaimsContextKey contextKey = "oidc_claims"
)

// ClaimsFromContext extracts validated token claims from the request context.
// Returns nil if no claims are present (e.g., auth not enabled or stdio mode).
func ClaimsFromContext(ctx context.Context) *TokenClaims {
	claims, _ := ctx.Value(ClaimsContextKey).(*TokenClaims)
	return claims
}

// AuthMiddlewareConfig configures the authentication middleware.
type AuthMiddlewareConfig struct {
	// OIDCValidator is the OIDC token validator. If nil, OIDC auth is disabled.
	OIDCValidator *OIDCValidator

	// StaticToken is a static bearer token for simple authentication.
	// This is the legacy HELM_MCP_AUTH_TOKEN mode.
	// If both OIDCValidator and StaticToken are set, OIDC takes precedence.
	StaticToken string

	// SessionCache is the optional session cache for validated tokens.
	SessionCache *SessionCache

	// AuditLogger logs security events. If nil, audit logging is disabled.
	AuditLogger *AuditLogger
}

// NewAuthMiddleware creates an HTTP middleware that validates bearer tokens.
//
// Authentication modes (in priority order):
//  1. OIDC JWT validation (if OIDCValidator is configured)
//  2. Static bearer token (if StaticToken is configured)
//  3. No auth (if neither is configured) — handler is returned as-is
//
// For OIDC mode, validated claims are injected into the request context
// and accessible via ClaimsFromContext().
func NewAuthMiddleware(config AuthMiddlewareConfig) func(http.Handler) http.Handler {
	// No auth configured — pass through.
	if config.OIDCValidator == nil && config.StaticToken == "" {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			// OIDC mode.
			if config.OIDCValidator != nil {
				token := extractBearerToken(authHeader)
				if token == "" {
					if config.AuditLogger != nil {
						config.AuditLogger.LogAuthFailure("missing bearer token", r.RemoteAddr)
					}
					http.Error(w, "Unauthorized: missing bearer token", http.StatusUnauthorized)
					return
				}

				// Check session cache first.
				var claims *TokenClaims
				cacheKey := ""
				if config.SessionCache != nil {
					// Use a hash of the token as a temporary cache key until we have claims.
					cacheKey = token // Will be replaced with proper key after validation.
					claims = config.SessionCache.Get(cacheKey)
				}

				if claims == nil {
					// Validate the token.
					var err error
					claims, err = config.OIDCValidator.ValidateToken(r.Context(), token)
					if err != nil {
						if config.AuditLogger != nil {
							config.AuditLogger.LogAuthFailure(err.Error(), r.RemoteAddr)
						}
						errMsg := err.Error()
						slog.Warn("OIDC token validation failed", "error", errMsg, "remote_addr", r.RemoteAddr) //nolint:gosec // G706: error from JWT library validation
						http.Error(w, "Unauthorized: "+errMsg, http.StatusUnauthorized)
						return
					}

					// Cache the validated claims with proper key.
					if config.SessionCache != nil {
						properKey := CacheKey(claims.ObjectID, claims.Subject)
						config.SessionCache.Put(properKey, claims)
					}
				}

				if config.AuditLogger != nil {
					config.AuditLogger.LogAuthSuccess(claims, "", r.RemoteAddr)
				}

				// Inject claims into request context.
				ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Static token mode (legacy HELM_MCP_AUTH_TOKEN).
			expected := []byte("Bearer " + config.StaticToken)
			if subtle.ConstantTimeCompare([]byte(authHeader), expected) != 1 {
				if config.AuditLogger != nil {
					config.AuditLogger.LogAuthFailure("invalid static token", r.RemoteAddr)
				}
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractBearerToken extracts the token from a "Bearer <token>" Authorization header.
func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

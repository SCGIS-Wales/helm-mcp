package security

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// --- extractBearerToken tests ---

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		header string
		want   string
	}{
		{"", ""},
		{"Bearer my-token-123", "my-token-123"},
		{"bearer my-token-123", "my-token-123"},
		{"BEARER my-token-123", "my-token-123"},
		{"Basic dXNlcjpwYXNz", ""},
		{"BearerNoSpace", ""},
		{"Bearer ", ""},
		{"Bearer   spaces-token  ", "spaces-token"},
	}

	for _, tt := range tests {
		got := extractBearerToken(tt.header)
		if got != tt.want {
			t.Errorf("extractBearerToken(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

// --- ClaimsFromContext tests ---

func TestClaimsFromContext_Present(t *testing.T) {
	claims := &TokenClaims{Subject: "user-123"}
	ctx := context.WithValue(context.Background(), ClaimsContextKey, claims)

	got := ClaimsFromContext(ctx)
	if got == nil {
		t.Fatal("expected non-nil claims")
	}
	if got.Subject != "user-123" {
		t.Errorf("Subject = %q", got.Subject)
	}
}

func TestClaimsFromContext_Absent(t *testing.T) {
	got := ClaimsFromContext(context.Background())
	if got != nil {
		t.Error("expected nil claims from empty context")
	}
}

// --- NewAuthMiddleware tests ---

func TestAuthMiddleware_NoAuth(t *testing.T) {
	// No auth configured — should pass through.
	middleware := NewAuthMiddleware(AuthMiddlewareConfig{})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestAuthMiddleware_StaticToken_Valid(t *testing.T) {
	middleware := NewAuthMiddleware(AuthMiddlewareConfig{
		StaticToken: "my-secret-token",
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer my-secret-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 for valid token", rec.Code)
	}
}

func TestAuthMiddleware_StaticToken_Invalid(t *testing.T) {
	middleware := NewAuthMiddleware(AuthMiddlewareConfig{
		StaticToken: "my-secret-token",
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for invalid token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for invalid token", rec.Code)
	}
}

func TestAuthMiddleware_StaticToken_Missing(t *testing.T) {
	middleware := NewAuthMiddleware(AuthMiddlewareConfig{
		StaticToken: "my-secret-token",
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for missing token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for missing token", rec.Code)
	}
}

func TestAuthMiddleware_OIDC_Valid(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	kid := "mw-test-key"

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"kid": kid,
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
				},
			},
		}
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL: "https://test-issuer.example.com",
		Audience:  "my-mcp-server",
		JWKSURL:   jwksServer.URL,
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	var buf bytes.Buffer
	auditLogger := NewAuditLogger(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))

	middleware := NewAuthMiddleware(AuthMiddlewareConfig{
		OIDCValidator: validator,
		AuditLogger:   auditLogger,
	})

	var receivedClaims *TokenClaims
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedClaims = ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Create a valid JWT.
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":                "https://test-issuer.example.com",
		"aud":                "my-mcp-server",
		"sub":                "user-456",
		"exp":                time.Now().Add(1 * time.Hour).Unix(),
		"oid":                "oid-789",
		"preferred_username": "test@example.com",
	})
	token.Header["kid"] = kid
	tokenString, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	if receivedClaims == nil {
		t.Fatal("expected claims in context")
	}
	if receivedClaims.Subject != "user-456" {
		t.Errorf("Subject = %q, want user-456", receivedClaims.Subject)
	}

	// Verify audit log was emitted.
	if output := buf.String(); output == "" {
		t.Error("expected audit log output")
	}
}

func TestAuthMiddleware_OIDC_MissingToken(t *testing.T) {
	validator, _ := NewOIDCValidator(OIDCConfig{
		IssuerURL: "https://test-issuer.example.com",
		Audience:  "my-mcp-server",
		JWKSURL:   "https://example.com/jwks",
	})

	middleware := NewAuthMiddleware(AuthMiddlewareConfig{
		OIDCValidator: validator,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuthMiddleware_OIDC_InvalidToken(t *testing.T) {
	validator, _ := NewOIDCValidator(OIDCConfig{
		IssuerURL: "https://test-issuer.example.com",
		Audience:  "my-mcp-server",
		JWKSURL:   "https://example.com/jwks",
	})

	middleware := NewAuthMiddleware(AuthMiddlewareConfig{
		OIDCValidator: validator,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAuthMiddleware_OIDC_WithSessionCache(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	kid := "cache-test-key"

	callCount := 0
	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256",
					"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
					"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
				},
			},
		}
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer jwksServer.Close()

	validator, _ := NewOIDCValidator(OIDCConfig{
		IssuerURL: "https://test-issuer.example.com",
		Audience:  "my-mcp-server",
		JWKSURL:   jwksServer.URL,
	})

	sessionCache := NewSessionCache(DefaultSessionConfig())
	defer sessionCache.Stop()

	middleware := NewAuthMiddleware(AuthMiddlewareConfig{
		OIDCValidator: validator,
		SessionCache:  sessionCache,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create token.
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": "https://test-issuer.example.com",
		"aud": "my-mcp-server",
		"sub": "user-cache",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"oid": "oid-cache",
	})
	token.Header["kid"] = kid
	tokenString, _ := token.SignedString(key)

	// First request — validates token, stores in cache.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("first request: status = %d, want 200", rec.Code)
	}

	// Verify session cache has an entry.
	if sessionCache.Size() != 1 {
		t.Errorf("expected 1 cache entry, got %d", sessionCache.Size())
	}
}

func TestAuthMiddleware_AuditLogging(t *testing.T) {
	var buf bytes.Buffer
	auditLogger := NewAuditLogger(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))

	middleware := NewAuthMiddleware(AuthMiddlewareConfig{
		StaticToken: "test-token",
		AuditLogger: auditLogger,
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Valid request — no audit failure.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Invalid request — should log auth_failure.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	output := buf.String()
	if output == "" {
		t.Error("expected audit log output for failed auth")
	}
}

package security

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// --- Test helpers ---

// testKeyPair holds an RSA key pair for testing.
type testKeyPair struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
}

func generateTestKeyPair(t *testing.T, kid string) *testKeyPair {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return &testKeyPair{
		privateKey: key,
		publicKey:  &key.PublicKey,
		kid:        kid,
	}
}

func (kp *testKeyPair) jwkJSON() map[string]interface{} {
	return map[string]interface{}{
		"kty": "RSA",
		"kid": kp.kid,
		"use": "sig",
		"alg": "RS256",
		"n":   base64.RawURLEncoding.EncodeToString(kp.publicKey.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(kp.publicKey.E)).Bytes()),
	}
}

func createTestJWT(t *testing.T, kp *testKeyPair, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kp.kid
	tokenString, err := token.SignedString(kp.privateKey)
	if err != nil {
		t.Fatalf("failed to sign test JWT: %v", err)
	}
	return tokenString
}

// setupJWKSServer creates a test HTTP server serving JWKS for the given key pairs.
func setupJWKSServer(t *testing.T, keys ...*testKeyPair) *httptest.Server {
	t.Helper()
	jwks := map[string]interface{}{
		"keys": func() []map[string]interface{} {
			result := make([]map[string]interface{}, len(keys))
			for i, kp := range keys {
				result[i] = kp.jwkJSON()
			}
			return result
		}(),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jwks":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(jwks)
		case "/.well-known/openid-configuration":
			baseURL := "http://" + r.Host
			doc := map[string]string{
				"issuer":   baseURL,
				"jwks_uri": baseURL + "/jwks",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(doc)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// --- OIDCConfig tests ---

func TestOIDCConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  OIDCConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  OIDCConfig{IssuerURL: "https://issuer.example.com", Audience: "my-app"},
			wantErr: false,
		},
		{
			name:    "missing issuer",
			config:  OIDCConfig{Audience: "my-app"},
			wantErr: true,
		},
		{
			name:    "missing audience",
			config:  OIDCConfig{IssuerURL: "https://issuer.example.com"},
			wantErr: true,
		},
		{
			name:    "both missing",
			config:  OIDCConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- NewOIDCValidator tests ---

func TestNewOIDCValidator_InvalidConfig(t *testing.T) {
	_, err := NewOIDCValidator(OIDCConfig{})
	if err == nil {
		t.Error("NewOIDCValidator with empty config should fail")
	}
}

func TestNewOIDCValidator_ValidConfig(t *testing.T) {
	v, err := NewOIDCValidator(OIDCConfig{
		IssuerURL: "https://issuer.example.com",
		Audience:  "my-app",
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator unexpected error: %v", err)
	}
	if v == nil {
		t.Fatal("expected non-nil validator")
	}
}

// --- ValidateToken tests ---

func TestValidateToken_Success(t *testing.T) {
	kp := generateTestKeyPair(t, "test-key-1")
	jwksServer := setupJWKSServer(t, kp)

	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL: jwksServer.URL,
		Audience:  "my-mcp-server",
		JWKSURL:   jwksServer.URL + "/jwks",
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	tokenString := createTestJWT(t, kp, jwt.MapClaims{
		"iss":                jwksServer.URL,
		"aud":                "my-mcp-server",
		"sub":                "user-123",
		"exp":                time.Now().Add(1 * time.Hour).Unix(),
		"iat":                time.Now().Unix(),
		"azp":                "client-app-1",
		"scp":                "helm.read helm.write",
		"roles":              []string{"HelmOperator"},
		"oid":                "00000000-0000-0000-0000-000000000001",
		"tid":                "tenant-abc",
		"preferred_username": "testuser@example.com",
		"uti":                "unique-token-id-xyz",
	})

	claims, err := validator.ValidateToken(context.Background(), tokenString)
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}

	// Verify extracted claims.
	if claims.Subject != "user-123" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "user-123")
	}
	if claims.AuthorizedParty != "client-app-1" {
		t.Errorf("AuthorizedParty = %q, want %q", claims.AuthorizedParty, "client-app-1")
	}
	if len(claims.Scopes) != 2 || claims.Scopes[0] != "helm.read" || claims.Scopes[1] != "helm.write" {
		t.Errorf("Scopes = %v, want [helm.read helm.write]", claims.Scopes)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "HelmOperator" {
		t.Errorf("Roles = %v, want [HelmOperator]", claims.Roles)
	}
	if claims.ObjectID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("ObjectID = %q, want %q", claims.ObjectID, "00000000-0000-0000-0000-000000000001")
	}
	if claims.TenantID != "tenant-abc" {
		t.Errorf("TenantID = %q, want %q", claims.TenantID, "tenant-abc")
	}
	if claims.PreferredUsername != "testuser@example.com" {
		t.Errorf("PreferredUsername = %q, want %q", claims.PreferredUsername, "testuser@example.com")
	}
	if claims.TokenID != "unique-token-id-xyz" {
		t.Errorf("TokenID = %q, want %q", claims.TokenID, "unique-token-id-xyz")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	kp := generateTestKeyPair(t, "test-key-1")
	jwksServer := setupJWKSServer(t, kp)

	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL: jwksServer.URL,
		Audience:  "my-mcp-server",
		JWKSURL:   jwksServer.URL + "/jwks",
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	tokenString := createTestJWT(t, kp, jwt.MapClaims{
		"iss": jwksServer.URL,
		"aud": "my-mcp-server",
		"sub": "user-123",
		"exp": time.Now().Add(-1 * time.Hour).Unix(), // Expired
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	})

	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err == nil {
		t.Error("ValidateToken should reject expired tokens")
	}
}

func TestValidateToken_WrongAudience(t *testing.T) {
	kp := generateTestKeyPair(t, "test-key-1")
	jwksServer := setupJWKSServer(t, kp)

	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL: jwksServer.URL,
		Audience:  "my-mcp-server",
		JWKSURL:   jwksServer.URL + "/jwks",
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	// Token issued for a different audience — must be rejected.
	tokenString := createTestJWT(t, kp, jwt.MapClaims{
		"iss": jwksServer.URL,
		"aud": "some-other-api", // Wrong audience
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err == nil {
		t.Error("ValidateToken should reject tokens with wrong audience (no token passthrough)")
	}
}

func TestValidateToken_WrongIssuer(t *testing.T) {
	kp := generateTestKeyPair(t, "test-key-1")
	jwksServer := setupJWKSServer(t, kp)

	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL: jwksServer.URL,
		Audience:  "my-mcp-server",
		JWKSURL:   jwksServer.URL + "/jwks",
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	tokenString := createTestJWT(t, kp, jwt.MapClaims{
		"iss": "https://evil-issuer.example.com", // Wrong issuer
		"aud": "my-mcp-server",
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err == nil {
		t.Error("ValidateToken should reject tokens with wrong issuer")
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	kp := generateTestKeyPair(t, "test-key-1")
	wrongKP := generateTestKeyPair(t, "test-key-1") // Same kid, different key
	jwksServer := setupJWKSServer(t, kp)

	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL: jwksServer.URL,
		Audience:  "my-mcp-server",
		JWKSURL:   jwksServer.URL + "/jwks",
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	// Sign with wrong key.
	tokenString := createTestJWT(t, wrongKP, jwt.MapClaims{
		"iss": jwksServer.URL,
		"aud": "my-mcp-server",
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err == nil {
		t.Error("ValidateToken should reject tokens with invalid signature")
	}
}

func TestValidateToken_UnknownKid(t *testing.T) {
	kp := generateTestKeyPair(t, "unknown-key")
	serverKP := generateTestKeyPair(t, "known-key")
	jwksServer := setupJWKSServer(t, serverKP)

	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL: jwksServer.URL,
		Audience:  "my-mcp-server",
		JWKSURL:   jwksServer.URL + "/jwks",
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	tokenString := createTestJWT(t, kp, jwt.MapClaims{
		"iss": jwksServer.URL,
		"aud": "my-mcp-server",
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err == nil {
		t.Error("ValidateToken should reject tokens with unknown kid")
	}
}

func TestValidateToken_AllowedClientIDs(t *testing.T) {
	kp := generateTestKeyPair(t, "test-key-1")
	jwksServer := setupJWKSServer(t, kp)

	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL:        jwksServer.URL,
		Audience:         "my-mcp-server",
		JWKSURL:          jwksServer.URL + "/jwks",
		AllowedClientIDs: []string{"allowed-client-1", "allowed-client-2"},
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	// Allowed client.
	tokenString := createTestJWT(t, kp, jwt.MapClaims{
		"iss": jwksServer.URL,
		"aud": "my-mcp-server",
		"sub": "user-123",
		"azp": "allowed-client-1",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err != nil {
		t.Errorf("ValidateToken should accept allowed client: %v", err)
	}

	// Disallowed client.
	tokenString = createTestJWT(t, kp, jwt.MapClaims{
		"iss": jwksServer.URL,
		"aud": "my-mcp-server",
		"sub": "user-123",
		"azp": "unauthorized-client",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err == nil {
		t.Error("ValidateToken should reject unauthorized client")
	}

	// Missing azp with AllowedClientIDs configured.
	tokenString = createTestJWT(t, kp, jwt.MapClaims{
		"iss": jwksServer.URL,
		"aud": "my-mcp-server",
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err == nil {
		t.Error("ValidateToken should reject tokens without azp when AllowedClientIDs is set")
	}
}

func TestValidateToken_RequiredScopes(t *testing.T) {
	kp := generateTestKeyPair(t, "test-key-1")
	jwksServer := setupJWKSServer(t, kp)

	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL:      jwksServer.URL,
		Audience:       "my-mcp-server",
		JWKSURL:        jwksServer.URL + "/jwks",
		RequiredScopes: []string{"helm.read", "helm.write"},
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	// Token with all required scopes.
	tokenString := createTestJWT(t, kp, jwt.MapClaims{
		"iss": jwksServer.URL,
		"aud": "my-mcp-server",
		"sub": "user-123",
		"scp": "helm.read helm.write helm.admin",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err != nil {
		t.Errorf("ValidateToken should accept token with all required scopes: %v", err)
	}

	// Token missing required scope.
	tokenString = createTestJWT(t, kp, jwt.MapClaims{
		"iss": jwksServer.URL,
		"aud": "my-mcp-server",
		"sub": "user-123",
		"scp": "helm.read", // Missing helm.write
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})
	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err == nil {
		t.Error("ValidateToken should reject token missing required scopes")
	}
}

func TestValidateToken_RequiredRoles(t *testing.T) {
	kp := generateTestKeyPair(t, "test-key-1")
	jwksServer := setupJWKSServer(t, kp)

	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL:     jwksServer.URL,
		Audience:      "my-mcp-server",
		JWKSURL:       jwksServer.URL + "/jwks",
		RequiredRoles: []string{"HelmOperator"},
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	// Token with required role.
	tokenString := createTestJWT(t, kp, jwt.MapClaims{
		"iss":   jwksServer.URL,
		"aud":   "my-mcp-server",
		"sub":   "user-123",
		"roles": []string{"HelmOperator", "Reader"},
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	})
	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err != nil {
		t.Errorf("ValidateToken should accept token with required role: %v", err)
	}

	// Token without required role.
	tokenString = createTestJWT(t, kp, jwt.MapClaims{
		"iss":   jwksServer.URL,
		"aud":   "my-mcp-server",
		"sub":   "user-123",
		"roles": []string{"Reader"},
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	})
	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err == nil {
		t.Error("ValidateToken should reject token without required role")
	}
}

func TestValidateToken_EntraIDAppIDClaim(t *testing.T) {
	// Entra ID v1 tokens use 'appid' instead of 'azp'.
	kp := generateTestKeyPair(t, "test-key-1")
	jwksServer := setupJWKSServer(t, kp)

	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL:        jwksServer.URL,
		Audience:         "my-mcp-server",
		JWKSURL:          jwksServer.URL + "/jwks",
		AllowedClientIDs: []string{"entra-app-id-123"},
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	tokenString := createTestJWT(t, kp, jwt.MapClaims{
		"iss":   jwksServer.URL,
		"aud":   "my-mcp-server",
		"sub":   "user-123",
		"appid": "entra-app-id-123", // Entra v1 format
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	})

	claims, err := validator.ValidateToken(context.Background(), tokenString)
	if err != nil {
		t.Fatalf("ValidateToken should accept Entra v1 appid claim: %v", err)
	}
	if claims.AuthorizedParty != "entra-app-id-123" {
		t.Errorf("AuthorizedParty = %q, want %q", claims.AuthorizedParty, "entra-app-id-123")
	}
}

func TestValidateToken_JWKSAutoDiscovery(t *testing.T) {
	kp := generateTestKeyPair(t, "test-key-1")
	jwksServer := setupJWKSServer(t, kp)

	// No JWKSURL — should auto-discover from issuer's .well-known/openid-configuration.
	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL: jwksServer.URL,
		Audience:  "my-mcp-server",
		// JWKSURL intentionally empty
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	tokenString := createTestJWT(t, kp, jwt.MapClaims{
		"iss": jwksServer.URL,
		"aud": "my-mcp-server",
		"sub": "user-123",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	})

	claims, err := validator.ValidateToken(context.Background(), tokenString)
	if err != nil {
		t.Fatalf("ValidateToken with JWKS auto-discovery error: %v", err)
	}
	if claims.Subject != "user-123" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "user-123")
	}
}

func TestValidateToken_NoExpirationClaim(t *testing.T) {
	kp := generateTestKeyPair(t, "test-key-1")
	jwksServer := setupJWKSServer(t, kp)

	validator, err := NewOIDCValidator(OIDCConfig{
		IssuerURL: jwksServer.URL,
		Audience:  "my-mcp-server",
		JWKSURL:   jwksServer.URL + "/jwks",
	})
	if err != nil {
		t.Fatalf("NewOIDCValidator error: %v", err)
	}

	// Token without exp claim.
	tokenString := createTestJWT(t, kp, jwt.MapClaims{
		"iss": jwksServer.URL,
		"aud": "my-mcp-server",
		"sub": "user-123",
	})

	_, err = validator.ValidateToken(context.Background(), tokenString)
	if err == nil {
		t.Error("ValidateToken should reject tokens without exp claim")
	}
}

// --- extractClaims tests ---

func TestExtractClaims_AudienceFormats(t *testing.T) {
	// String audience.
	claims := extractClaims(jwt.MapClaims{"aud": "single-aud"})
	if len(claims.Audience) != 1 || claims.Audience[0] != "single-aud" {
		t.Errorf("string aud: got %v", claims.Audience)
	}

	// Array audience.
	claims = extractClaims(jwt.MapClaims{"aud": []interface{}{"aud-1", "aud-2"}})
	if len(claims.Audience) != 2 {
		t.Errorf("array aud: got %v", claims.Audience)
	}
}

func TestExtractClaims_JTIAndUTI(t *testing.T) {
	// jti claim
	claims := extractClaims(jwt.MapClaims{"jti": "jti-123"})
	if claims.TokenID != "jti-123" {
		t.Errorf("jti: TokenID = %q", claims.TokenID)
	}

	// uti claim (Entra ID)
	claims = extractClaims(jwt.MapClaims{"uti": "uti-456"})
	if claims.TokenID != "uti-456" {
		t.Errorf("uti: TokenID = %q", claims.TokenID)
	}

	// jti takes precedence
	claims = extractClaims(jwt.MapClaims{"jti": "jti-123", "uti": "uti-456"})
	if claims.TokenID != "jti-123" {
		t.Errorf("jti+uti: TokenID = %q, want jti-123", claims.TokenID)
	}
}

// --- validateRequiredStrings tests ---

func TestValidateRequiredStrings(t *testing.T) {
	// All present.
	err := validateRequiredStrings("scope", []string{"a", "b", "c"}, []string{"a", "b"})
	if err != nil {
		t.Errorf("all present: unexpected error: %v", err)
	}

	// Missing one.
	err = validateRequiredStrings("scope", []string{"a"}, []string{"a", "b"})
	if err == nil {
		t.Error("missing: expected error")
	}

	// Empty actual.
	err = validateRequiredStrings("role", nil, []string{"admin"})
	if err == nil {
		t.Error("empty actual: expected error")
	}

	// Empty required.
	err = validateRequiredStrings("scope", []string{"a"}, nil)
	if err != nil {
		t.Errorf("empty required: unexpected error: %v", err)
	}
}

// --- JWKS parsing tests ---

func TestParseRSAPublicKey(t *testing.T) {
	kp := generateTestKeyPair(t, "test")

	key := jwkKey{
		Kty: "RSA",
		Kid: "test",
		Use: "sig",
		N:   base64.RawURLEncoding.EncodeToString(kp.publicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(kp.publicKey.E)).Bytes()),
	}

	parsed, err := parseRSAPublicKey(key)
	if err != nil {
		t.Fatalf("parseRSAPublicKey error: %v", err)
	}
	if parsed.N.Cmp(kp.publicKey.N) != 0 {
		t.Error("parsed modulus doesn't match")
	}
	if parsed.E != kp.publicKey.E {
		t.Error("parsed exponent doesn't match")
	}
}

func TestParseRSAPublicKey_InvalidModulus(t *testing.T) {
	key := jwkKey{N: "!!!invalid-base64!!!", E: "AQAB"}
	_, err := parseRSAPublicKey(key)
	if err == nil {
		t.Error("expected error for invalid modulus encoding")
	}
}

func TestParseRSAPublicKey_InvalidExponent(t *testing.T) {
	kp := generateTestKeyPair(t, "test")
	key := jwkKey{
		N: base64.RawURLEncoding.EncodeToString(kp.publicKey.N.Bytes()),
		E: "!!!invalid!!!",
	}
	_, err := parseRSAPublicKey(key)
	if err == nil {
		t.Error("expected error for invalid exponent encoding")
	}
}

// --- JWKS fetch tests ---

func TestFetchJWKS_ValidResponse(t *testing.T) {
	kp := generateTestKeyPair(t, "key-1")
	jwksServer := setupJWKSServer(t, kp)

	keys, err := fetchJWKS(context.Background(), jwksServer.URL+"/jwks", http.DefaultClient)
	if err != nil {
		t.Fatalf("fetchJWKS error: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("expected 1 key, got %d", len(keys))
	}
	if _, ok := keys["key-1"]; !ok {
		t.Error("expected key with kid 'key-1'")
	}
}

func TestFetchJWKS_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"keys":[]}`)
	}))
	defer server.Close()

	_, err := fetchJWKS(context.Background(), server.URL, http.DefaultClient)
	if err == nil {
		t.Error("expected error for empty JWKS")
	}
}

func TestFetchJWKS_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := fetchJWKS(context.Background(), server.URL, http.DefaultClient)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestFetchJWKS_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `not-json`)
	}))
	defer server.Close()

	_, err := fetchJWKS(context.Background(), server.URL, http.DefaultClient)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// --- JWKS cache tests ---

func TestJWKSCache_CachesKeys(t *testing.T) {
	kp := generateTestKeyPair(t, "key-1")
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{kp.jwkJSON()},
		}
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cache := newJWKSCache()
	ctx := context.Background()

	// First call should hit the server.
	_, err := cache.GetKeys(ctx, server.URL, http.DefaultClient)
	if err != nil {
		t.Fatalf("first GetKeys error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 server call, got %d", callCount)
	}

	// Second call should use cache.
	_, err = cache.GetKeys(ctx, server.URL, http.DefaultClient)
	if err != nil {
		t.Fatalf("second GetKeys error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected cache hit (still 1 call), got %d", callCount)
	}
}

func TestJWKSCache_RefreshKeys(t *testing.T) {
	kp := generateTestKeyPair(t, "key-1")
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{kp.jwkJSON()},
		}
		_ = json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	cache := newJWKSCache()
	ctx := context.Background()

	_, _ = cache.GetKeys(ctx, server.URL, http.DefaultClient)
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}

	// RefreshKeys with a fresh cache should be a no-op (double-check locking).
	_, err := cache.RefreshKeys(ctx, server.URL, http.DefaultClient)
	if err != nil {
		t.Fatalf("RefreshKeys error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call (double-check should skip re-fetch), got %d", callCount)
	}

	// Expire the cache and verify RefreshKeys actually fetches.
	cache.mu.Lock()
	cache.fetched = time.Now().Add(-2 * time.Hour)
	cache.mu.Unlock()

	_, err = cache.RefreshKeys(ctx, server.URL, http.DefaultClient)
	if err != nil {
		t.Fatalf("RefreshKeys after expiry error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls after cache expiry, got %d", callCount)
	}
}

// Package security provides OIDC/OAuth2 token validation for the Helm MCP server.
//
// This implements MCP authorization requirements:
//   - JWT validation with JWKS (signature, issuer, audience, expiry, azp)
//   - No acceptance of tokens issued for other resources
//   - Configurable via environment variables
package security

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// OIDCConfig holds the configuration for OIDC/OAuth2 token validation.
type OIDCConfig struct {
	// IssuerURL is the expected token issuer (iss claim).
	// Example: "https://login.microsoftonline.com/{tenant}/v2.0"
	IssuerURL string

	// Audience is the expected audience (aud claim) — must match this MCP server's app ID.
	// Tokens issued for other resources are rejected.
	Audience string

	// JWKSURL is the URL to fetch JSON Web Key Sets for signature verification.
	// If empty, it is auto-discovered from IssuerURL + "/.well-known/openid-configuration".
	JWKSURL string

	// RequiredScopes is a list of OAuth2 scopes that must be present in the token's scp claim.
	RequiredScopes []string

	// RequiredRoles is a list of app roles that must be present in the token's roles claim.
	RequiredRoles []string

	// AllowedClientIDs restricts which client applications (azp/appid) may call this server.
	// If empty, any client with a valid token is allowed.
	AllowedClientIDs []string

	// HTTPClient is used for JWKS and discovery fetches. If nil, http.DefaultClient is used.
	HTTPClient *http.Client
}

// Validate checks that required configuration fields are set.
func (c *OIDCConfig) Validate() error {
	if c.IssuerURL == "" {
		return fmt.Errorf("OIDC issuer URL is required")
	}
	if c.Audience == "" {
		return fmt.Errorf("OIDC audience is required")
	}
	return nil
}

// TokenClaims contains the validated claims extracted from an OIDC token.
type TokenClaims struct {
	// Subject is the user principal (sub claim).
	Subject string
	// Issuer is the token issuer (iss claim).
	Issuer string
	// Audience contains the audience(s) the token was issued for.
	Audience []string
	// ExpiresAt is the token expiration time.
	ExpiresAt time.Time
	// IssuedAt is the token issuance time.
	IssuedAt time.Time
	// AuthorizedParty is the azp/appid claim — the client application that requested the token.
	AuthorizedParty string
	// Scopes contains the delegated permission scopes (scp claim, space-separated in Entra ID).
	Scopes []string
	// Roles contains the app roles assigned to the user (roles claim).
	Roles []string
	// ObjectID is the Entra ID object identifier (oid claim).
	ObjectID string
	// TenantID is the Entra ID tenant (tid claim).
	TenantID string
	// PreferredUsername is the user's display name / UPN.
	PreferredUsername string
	// TokenID is a unique identifier for the token (jti/uti claim).
	TokenID string
}

// OIDCValidator validates OAuth2/OIDC JWT tokens using JWKS.
type OIDCValidator struct {
	config  OIDCConfig
	jwks    *jwksCache
	httpCli *http.Client
}

// NewOIDCValidator creates a new OIDC token validator.
func NewOIDCValidator(config OIDCConfig) (*OIDCValidator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid OIDC config: %w", err)
	}

	httpCli := config.HTTPClient
	if httpCli == nil {
		httpCli = &http.Client{Timeout: 10 * time.Second}
	}

	v := &OIDCValidator{
		config:  config,
		httpCli: httpCli,
		jwks:    newJWKSCache(),
	}

	return v, nil
}

// ValidateToken validates a JWT bearer token and returns the extracted claims.
// It verifies: signature (via JWKS), issuer, audience, expiry, and authorized party.
func (v *OIDCValidator) ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	// Fetch (or use cached) JWKS keys.
	jwksURL := v.config.JWKSURL
	if jwksURL == "" {
		discovered, err := v.discoverJWKS(ctx)
		if err != nil {
			return nil, fmt.Errorf("JWKS discovery failed: %w", err)
		}
		jwksURL = discovered
	}

	keys, err := v.jwks.GetKeys(ctx, jwksURL, v.httpCli)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	// Parse and validate the JWT.
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing algorithm is RSA (Entra ID uses RS256).
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("token missing kid header")
		}

		key, found := keys[kid]
		if !found {
			// Key not found — try refreshing JWKS (key rotation).
			refreshedKeys, refreshErr := v.jwks.RefreshKeys(ctx, jwksURL, v.httpCli)
			if refreshErr != nil {
				return nil, fmt.Errorf("JWKS refresh failed: %w", refreshErr)
			}
			key, found = refreshedKeys[kid]
			if !found {
				return nil, fmt.Errorf("signing key %q not found in JWKS", kid)
			}
		}

		return key, nil
	},
		jwt.WithIssuer(v.config.IssuerURL),
		jwt.WithAudience(v.config.Audience),
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}),
	)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to extract token claims")
	}

	claims := extractClaims(mapClaims)

	// Validate authorized party (azp/appid) if AllowedClientIDs is configured.
	if len(v.config.AllowedClientIDs) > 0 {
		if claims.AuthorizedParty == "" {
			return nil, fmt.Errorf("token missing azp/appid claim")
		}
		allowed := false
		for _, id := range v.config.AllowedClientIDs {
			if claims.AuthorizedParty == id {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("client %q is not in the allowed list", claims.AuthorizedParty)
		}
	}

	// Validate required scopes.
	if len(v.config.RequiredScopes) > 0 {
		if err := validateRequiredStrings("scope", claims.Scopes, v.config.RequiredScopes); err != nil {
			return nil, err
		}
	}

	// Validate required roles.
	if len(v.config.RequiredRoles) > 0 {
		if err := validateRequiredStrings("role", claims.Roles, v.config.RequiredRoles); err != nil {
			return nil, err
		}
	}

	return claims, nil
}

// discoverJWKS fetches the JWKS URL from the OIDC discovery endpoint.
func (v *OIDCValidator) discoverJWKS(ctx context.Context) (string, error) {
	discoveryURL := strings.TrimSuffix(v.config.IssuerURL, "/") + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating discovery request: %w", err)
	}

	resp, err := v.httpCli.Do(req) //nolint:gosec // discovery URL is derived from trusted IssuerURL config
	if err != nil {
		return "", fmt.Errorf("fetching discovery document: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("discovery endpoint returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return "", fmt.Errorf("reading discovery document: %w", err)
	}

	var doc struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", fmt.Errorf("parsing discovery document: %w", err)
	}
	if doc.JWKSURI == "" {
		return "", fmt.Errorf("discovery document missing jwks_uri")
	}

	return doc.JWKSURI, nil
}

// extractClaims extracts typed claims from a JWT MapClaims.
func extractClaims(m jwt.MapClaims) *TokenClaims {
	c := &TokenClaims{}

	if sub, ok := m["sub"].(string); ok {
		c.Subject = sub
	}
	if iss, ok := m["iss"].(string); ok {
		c.Issuer = iss
	}
	// aud can be string or []string
	switch aud := m["aud"].(type) {
	case string:
		c.Audience = []string{aud}
	case []interface{}:
		for _, a := range aud {
			if s, ok := a.(string); ok {
				c.Audience = append(c.Audience, s)
			}
		}
	}
	if exp, ok := m["exp"].(float64); ok {
		c.ExpiresAt = time.Unix(int64(exp), 0)
	}
	if iat, ok := m["iat"].(float64); ok {
		c.IssuedAt = time.Unix(int64(iat), 0)
	}
	// azp (OIDC) or appid (Entra v1)
	if azp, ok := m["azp"].(string); ok {
		c.AuthorizedParty = azp
	} else if appid, ok := m["appid"].(string); ok {
		c.AuthorizedParty = appid
	}
	// scp (space-separated scopes in Entra v2)
	if scp, ok := m["scp"].(string); ok {
		c.Scopes = strings.Fields(scp)
	}
	// roles (array of app roles)
	if roles, ok := m["roles"].([]interface{}); ok {
		for _, r := range roles {
			if s, ok := r.(string); ok {
				c.Roles = append(c.Roles, s)
			}
		}
	}
	if oid, ok := m["oid"].(string); ok {
		c.ObjectID = oid
	}
	if tid, ok := m["tid"].(string); ok {
		c.TenantID = tid
	}
	if upn, ok := m["preferred_username"].(string); ok {
		c.PreferredUsername = upn
	}
	// jti or uti (Entra uses uti for unique token identifier)
	if jti, ok := m["jti"].(string); ok {
		c.TokenID = jti
	} else if uti, ok := m["uti"].(string); ok {
		c.TokenID = uti
	}

	return c
}

// validateRequiredStrings checks that all required values are present in the actual set.
func validateRequiredStrings(kind string, actual, required []string) error {
	have := make(map[string]bool, len(actual))
	for _, s := range actual {
		have[s] = true
	}
	var missing []string
	for _, r := range required {
		if !have[r] {
			missing = append(missing, r)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required %s(s): %s", kind, strings.Join(missing, ", "))
	}
	return nil
}

// --- JWKS Cache ---

// jwksCache provides thread-safe caching of JWKS keys with automatic refresh.
type jwksCache struct {
	mu      sync.RWMutex
	keys    map[string]*rsa.PublicKey
	url     string
	fetched time.Time
	ttl     time.Duration
}

func newJWKSCache() *jwksCache {
	return &jwksCache{
		ttl: 1 * time.Hour, // JWKS keys are rotated infrequently; 1h cache is standard.
	}
}

// GetKeys returns cached JWKS keys, fetching them if the cache is empty or expired.
func (c *jwksCache) GetKeys(ctx context.Context, jwksURL string, httpCli *http.Client) (map[string]*rsa.PublicKey, error) {
	c.mu.RLock()
	if c.url == jwksURL && time.Since(c.fetched) < c.ttl && len(c.keys) > 0 {
		keys := c.keys
		c.mu.RUnlock()
		return keys, nil
	}
	c.mu.RUnlock()

	return c.RefreshKeys(ctx, jwksURL, httpCli)
}

// RefreshKeys forces a fetch of JWKS keys and updates the cache.
// Uses double-check locking to avoid thundering herd on concurrent cache misses.
func (c *jwksCache) RefreshKeys(ctx context.Context, jwksURL string, httpCli *http.Client) (map[string]*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check: another goroutine may have already refreshed while we waited for the lock.
	if c.url == jwksURL && time.Since(c.fetched) < c.ttl && len(c.keys) > 0 {
		return c.keys, nil
	}

	keys, err := fetchJWKS(ctx, jwksURL, httpCli)
	if err != nil {
		return nil, err
	}

	c.keys = keys
	c.url = jwksURL
	c.fetched = time.Now()

	slog.Debug("JWKS keys refreshed", "url", jwksURL, "keys", len(keys)) //nolint:gosec // G706: jwksURL is from trusted OIDC configuration, not user input
	return keys, nil
}

// jwksResponse represents the JSON Web Key Set response.
type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	Alg string `json:"alg"`
}

// fetchJWKS fetches and parses JWKS from the given URL.
func fetchJWKS(ctx context.Context, jwksURL string, httpCli *http.Client) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating JWKS request: %w", err)
	}

	resp, err := httpCli.Do(req) //nolint:gosec // JWKS URL is from trusted OIDC configuration
	if err != nil {
		return nil, fmt.Errorf("fetching JWKS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, fmt.Errorf("reading JWKS response: %w", err)
	}

	var jwks jwksResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, fmt.Errorf("parsing JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, k := range jwks.Keys {
		if k.Use != "sig" {
			continue
		}
		if k.Kty != "RSA" {
			slog.Warn("skipping non-RSA signing key in JWKS (only RSA is supported)", "kid", k.Kid, "kty", k.Kty)
			continue
		}
		pubKey, err := parseRSAPublicKey(k)
		if err != nil {
			slog.Warn("skipping malformed JWKS key", "kid", k.Kid, "error", err)
			continue
		}
		keys[k.Kid] = pubKey
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no valid RSA signing keys found in JWKS")
	}

	return keys, nil
}

// parseRSAPublicKey constructs an RSA public key from JWK parameters.
func parseRSAPublicKey(k jwkKey) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decoding modulus: %w", err)
	}
	// Enforce minimum 2048-bit RSA key size (256 bytes) per CA/B Forum requirements.
	if len(nBytes) < 256 {
		return nil, fmt.Errorf("RSA key too small: %d bits (minimum 2048)", len(nBytes)*8)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decoding exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

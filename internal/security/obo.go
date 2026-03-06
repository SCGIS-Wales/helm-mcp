package security

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OBOConfig configures the On-Behalf-Of token exchange for downstream API calls.
//
// Per MCP Security Best Practices:
//   - MCP servers must NOT forward user tokens to downstream APIs (no passthrough).
//   - When user context is needed downstream, OBO token exchange acquires a new
//     access token with aud = downstream API.
//   - Tokens must be scoped only for the target downstream resource.
type OBOConfig struct {
	// TokenURL is the Entra ID token endpoint.
	// Example: "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token"
	TokenURL string

	// ClientID is this MCP server's registered application (client) ID.
	ClientID string

	// ClientSecret is this MCP server's client secret for confidential client auth.
	ClientSecret string

	// HTTPClient is used for token exchange requests. If nil, a default client is used.
	HTTPClient *http.Client
}

// Validate checks that required OBO configuration fields are set.
func (c *OBOConfig) Validate() error {
	if c.TokenURL == "" {
		return fmt.Errorf("OBO token URL is required")
	}
	if c.ClientID == "" {
		return fmt.Errorf("OBO client ID is required")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("OBO client secret is required")
	}
	return nil
}

// OBOTokenResponse represents a successful token exchange response.
type OBOTokenResponse struct {
	// AccessToken is the new token scoped for the downstream API.
	AccessToken string `json:"access_token"`
	// TokenType is typically "Bearer".
	TokenType string `json:"token_type"`
	// ExpiresIn is the token lifetime in seconds.
	ExpiresIn int `json:"expires_in"`
	// Scope contains the granted scopes.
	Scope string `json:"scope"`
	// ExpiresAt is the calculated expiry time (not from the response).
	ExpiresAt time.Time `json:"-"`
}

// OBOExchanger performs On-Behalf-Of token exchanges with Entra ID.
type OBOExchanger struct {
	config  OBOConfig
	httpCli *http.Client
}

// NewOBOExchanger creates a new OBO token exchanger.
func NewOBOExchanger(config OBOConfig) (*OBOExchanger, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid OBO config: %w", err)
	}

	httpCli := config.HTTPClient
	if httpCli == nil {
		httpCli = &http.Client{Timeout: 10 * time.Second}
	}

	return &OBOExchanger{
		config:  config,
		httpCli: httpCli,
	}, nil
}

// Exchange performs an OBO token exchange, acquiring a new token for the
// specified downstream resource on behalf of the user identified by the
// incoming assertion token.
//
// Per the Entra ID OBO flow:
//
//	grant_type       = urn:ietf:params:oauth:grant-type:jwt-bearer
//	requested_token_use = on_behalf_of
//	assertion        = incoming user token
//	scope            = scopes for downstream API
//
// The returned token has aud = downstream API, preserving user identity in claims.
func (e *OBOExchanger) Exchange(ctx context.Context, assertion string, downstreamScopes []string) (*OBOTokenResponse, error) {
	if assertion == "" {
		return nil, fmt.Errorf("assertion token is required for OBO exchange")
	}
	if len(downstreamScopes) == 0 {
		return nil, fmt.Errorf("at least one downstream scope is required")
	}

	// Hardcoded token endpoint URL — never constructed from untrusted input.
	// This mitigates SSRF risk at the OBO token endpoint (per technical assessment).
	form := url.Values{
		"grant_type":          {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"client_id":           {e.config.ClientID},
		"client_secret":       {e.config.ClientSecret},
		"assertion":           {assertion},
		"scope":               {strings.Join(downstreamScopes, " ")},
		"requested_token_use": {"on_behalf_of"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.config.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating OBO request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	start := time.Now()
	resp, err := e.httpCli.Do(req) //nolint:gosec // TokenURL is from trusted server configuration, not user input
	duration := time.Since(start)
	if err != nil {
		scrubbed := ScrubError(err).Error()
		slog.Error("OBO token exchange failed", //nolint:gosec // error is scrubbed before logging
			"duration_ms", duration.Milliseconds(),
			"error", scrubbed,
		)
		return nil, fmt.Errorf("OBO token exchange request failed: %w", ScrubError(err))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, fmt.Errorf("reading OBO response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Parse error response for diagnostics.
		var errResp struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
			Codes       []int  `json:"error_codes"`
			CorrelID    string `json:"correlation_id"`
			TraceID     string `json:"trace_id"`
			Claims      string `json:"claims"`
		}
		_ = json.Unmarshal(body, &errResp)

		slog.Error("OBO token exchange error", //nolint:gosec // error fields are from Entra ID token endpoint response
			"status", resp.StatusCode,
			"error", errResp.Error,
			"error_codes", errResp.Codes,
			"correlation_id", errResp.CorrelID,
			"trace_id", errResp.TraceID,
			"duration_ms", duration.Milliseconds(),
		)

		return nil, classifyOBOError(resp.StatusCode, errResp.Error, errResp.Description)
	}

	var tokenResp OBOTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing OBO token response: %w", err)
	}

	tokenResp.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	slog.Debug("OBO token exchange successful",
		"scope", tokenResp.Scope,
		"expires_in", tokenResp.ExpiresIn,
		"duration_ms", duration.Milliseconds(),
	)

	return &tokenResp, nil
}

// classifyOBOError maps Entra ID error codes to actionable error messages.
// See: https://learn.microsoft.com/en-us/entra/identity-platform/v2-oauth2-on-behalf-of-flow#error-response
func classifyOBOError(status int, errCode, description string) error {
	switch errCode {
	case "interaction_required":
		// Conditional Access policy requires user interaction (MFA, consent, etc.).
		// The caller must propagate the claims challenge back to the client.
		return &OBOError{Code: errCode, Description: description, Retryable: false,
			Guidance: "user must re-authenticate; propagate claims challenge to the client"}
	case "consent_required":
		// The downstream API requires admin or user consent that has not been granted.
		return &OBOError{Code: errCode, Description: description, Retryable: false,
			Guidance: "admin consent required for the downstream API scope; run admin consent flow in Azure portal"}
	case "invalid_grant":
		// The assertion token is expired, revoked, or malformed.
		return &OBOError{Code: errCode, Description: description, Retryable: false,
			Guidance: "assertion token is invalid or expired; the user may need to sign in again"}
	case "invalid_scope":
		// The requested scope is not registered or recognized by the downstream API.
		return &OBOError{Code: errCode, Description: description, Retryable: false,
			Guidance: "check that the downstream scope is registered in the app registration's 'API permissions'"}
	case "invalid_client":
		// Client authentication failed (wrong client_id or client_secret).
		return &OBOError{Code: errCode, Description: description, Retryable: false,
			Guidance: "verify HELM_MCP_OBO_CLIENT_ID and HELM_MCP_OBO_CLIENT_SECRET are correct"}
	case "temporarily_unavailable":
		// Entra ID is experiencing transient issues.
		return &OBOError{Code: errCode, Description: description, Retryable: true,
			Guidance: "Entra ID is temporarily unavailable; retry with exponential backoff"}
	default:
		return fmt.Errorf("OBO exchange failed (HTTP %d): %s — %s", status, errCode, description)
	}
}

// OBOError represents a classified OBO token exchange error with actionable guidance.
type OBOError struct {
	Code        string
	Description string
	Retryable   bool
	Guidance    string
}

func (e *OBOError) Error() string {
	return fmt.Sprintf("OBO exchange error [%s]: %s (hint: %s)", e.Code, e.Description, e.Guidance)
}

// IsRetryable returns true if the error is transient and the operation can be retried.
func (e *OBOError) IsRetryable() bool {
	return e.Retryable
}

package security

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- OBOConfig tests ---

func TestOBOConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  OBOConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: OBOConfig{
				TokenURL:     "https://login.microsoftonline.com/tenant/oauth2/v2.0/token",
				ClientID:     "my-client-id",
				ClientSecret: "my-client-secret",
			},
			wantErr: false,
		},
		{
			name: "missing token URL",
			config: OBOConfig{
				ClientID:     "my-client-id",
				ClientSecret: "my-client-secret",
			},
			wantErr: true,
		},
		{
			name: "missing client ID",
			config: OBOConfig{
				TokenURL:     "https://login.microsoftonline.com/tenant/oauth2/v2.0/token",
				ClientSecret: "my-client-secret",
			},
			wantErr: true,
		},
		{
			name: "missing client secret",
			config: OBOConfig{
				TokenURL: "https://login.microsoftonline.com/tenant/oauth2/v2.0/token",
				ClientID: "my-client-id",
			},
			wantErr: true,
		},
		{
			name:    "all missing",
			config:  OBOConfig{},
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

// --- NewOBOExchanger tests ---

func TestNewOBOExchanger_InvalidConfig(t *testing.T) {
	_, err := NewOBOExchanger(OBOConfig{})
	if err == nil {
		t.Error("NewOBOExchanger with empty config should fail")
	}
}

func TestNewOBOExchanger_ValidConfig(t *testing.T) {
	e, err := NewOBOExchanger(OBOConfig{
		TokenURL:     "https://login.microsoftonline.com/tenant/oauth2/v2.0/token",
		ClientID:     "my-client-id",
		ClientSecret: "my-client-secret",
	})
	if err != nil {
		t.Fatalf("NewOBOExchanger unexpected error: %v", err)
	}
	if e == nil {
		t.Fatal("expected non-nil exchanger")
	}
}

// --- Exchange tests ---

func TestOBOExchange_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("expected form content type, got %s", ct)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}

		// Verify OBO grant type.
		if gt := r.FormValue("grant_type"); gt != "urn:ietf:params:oauth:grant-type:jwt-bearer" {
			t.Errorf("wrong grant_type: %s", gt)
		}
		if rtu := r.FormValue("requested_token_use"); rtu != "on_behalf_of" {
			t.Errorf("wrong requested_token_use: %s", rtu)
		}
		if a := r.FormValue("assertion"); a != "incoming-user-token" {
			t.Errorf("wrong assertion: %s", a)
		}
		if s := r.FormValue("scope"); s != "api://downstream/.default" {
			t.Errorf("wrong scope: %s", s)
		}
		if cid := r.FormValue("client_id"); cid != "my-client-id" {
			t.Errorf("wrong client_id: %s", cid)
		}

		resp := map[string]interface{}{
			"access_token": "new-obo-token-for-downstream",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"scope":        "api://downstream/.default",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	exchanger, err := NewOBOExchanger(OBOConfig{
		TokenURL:     server.URL,
		ClientID:     "my-client-id",
		ClientSecret: "my-secret",
	})
	if err != nil {
		t.Fatalf("NewOBOExchanger error: %v", err)
	}

	resp, err := exchanger.Exchange(
		context.Background(),
		"incoming-user-token",
		[]string{"api://downstream/.default"},
	)
	if err != nil {
		t.Fatalf("Exchange error: %v", err)
	}

	if resp.AccessToken != "new-obo-token-for-downstream" {
		t.Errorf("AccessToken = %q", resp.AccessToken)
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("TokenType = %q", resp.TokenType)
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d", resp.ExpiresIn)
	}
	if resp.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set")
	}
}

func TestOBOExchange_EmptyAssertion(t *testing.T) {
	exchanger, _ := NewOBOExchanger(OBOConfig{
		TokenURL:     "https://example.com/token",
		ClientID:     "id",
		ClientSecret: "secret",
	})

	_, err := exchanger.Exchange(context.Background(), "", []string{"scope"})
	if err == nil {
		t.Error("Exchange with empty assertion should fail")
	}
}

func TestOBOExchange_EmptyScopes(t *testing.T) {
	exchanger, _ := NewOBOExchanger(OBOConfig{
		TokenURL:     "https://example.com/token",
		ClientID:     "id",
		ClientSecret: "secret",
	})

	_, err := exchanger.Exchange(context.Background(), "token", nil)
	if err == nil {
		t.Error("Exchange with empty scopes should fail")
	}
}

func TestOBOExchange_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "server_error",
			"error_description": "internal failure",
		})
	}))
	defer server.Close()

	exchanger, _ := NewOBOExchanger(OBOConfig{
		TokenURL:     server.URL,
		ClientID:     "id",
		ClientSecret: "secret",
	})

	_, err := exchanger.Exchange(context.Background(), "token", []string{"scope"})
	if err == nil {
		t.Error("Exchange should fail on server error")
	}
}

func TestOBOExchange_InteractionRequired(t *testing.T) {
	// Conditional Access re-evaluation returns interaction_required.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "interaction_required",
			"error_description": "AADSTS50076: MFA required",
		})
	}))
	defer server.Close()

	exchanger, _ := NewOBOExchanger(OBOConfig{
		TokenURL:     server.URL,
		ClientID:     "id",
		ClientSecret: "secret",
	})

	_, err := exchanger.Exchange(context.Background(), "token", []string{"scope"})
	if err == nil {
		t.Fatal("Exchange should fail on interaction_required")
	}

	oboErr, ok := err.(*OBOError)
	if !ok {
		t.Fatalf("expected *OBOError, got %T: %v", err, err)
	}
	if oboErr.Code != "interaction_required" {
		t.Errorf("Code = %q, want interaction_required", oboErr.Code)
	}
	if oboErr.Retryable {
		t.Error("interaction_required should not be retryable")
	}
}

func TestOBOExchange_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer server.Close()

	exchanger, _ := NewOBOExchanger(OBOConfig{
		TokenURL:     server.URL,
		ClientID:     "id",
		ClientSecret: "secret",
	})

	_, err := exchanger.Exchange(context.Background(), "token", []string{"scope"})
	if err == nil {
		t.Error("Exchange should fail on invalid JSON response")
	}
}

func TestOBOExchange_ConsentRequired(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":             "consent_required",
			"error_description": "AADSTS65001: user needs to consent",
			"error_codes":       []int{65001},
			"correlation_id":    "abc-123",
			"trace_id":          "trace-456",
		})
	}))
	defer server.Close()

	exchanger, _ := NewOBOExchanger(OBOConfig{
		TokenURL:     server.URL,
		ClientID:     "id",
		ClientSecret: "secret",
	})

	_, err := exchanger.Exchange(context.Background(), "token", []string{"scope"})
	if err == nil {
		t.Fatal("Exchange should fail on consent_required")
	}

	oboErr, ok := err.(*OBOError)
	if !ok {
		t.Fatalf("expected *OBOError, got %T: %v", err, err)
	}
	if oboErr.Code != "consent_required" {
		t.Errorf("Code = %q, want consent_required", oboErr.Code)
	}
	if oboErr.Retryable {
		t.Error("consent_required should not be retryable")
	}
}

func TestOBOExchange_InvalidGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "AADSTS70000: assertion expired",
		})
	}))
	defer server.Close()

	exchanger, _ := NewOBOExchanger(OBOConfig{
		TokenURL:     server.URL,
		ClientID:     "id",
		ClientSecret: "secret",
	})

	_, err := exchanger.Exchange(context.Background(), "token", []string{"scope"})
	if err == nil {
		t.Fatal("Exchange should fail on invalid_grant")
	}

	oboErr, ok := err.(*OBOError)
	if !ok {
		t.Fatalf("expected *OBOError, got %T", err)
	}
	if oboErr.Code != "invalid_grant" {
		t.Errorf("Code = %q, want invalid_grant", oboErr.Code)
	}
}

func TestOBOExchange_TemporarilyUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "temporarily_unavailable",
			"error_description": "service is overloaded",
		})
	}))
	defer server.Close()

	exchanger, _ := NewOBOExchanger(OBOConfig{
		TokenURL:     server.URL,
		ClientID:     "id",
		ClientSecret: "secret",
	})

	_, err := exchanger.Exchange(context.Background(), "token", []string{"scope"})
	if err == nil {
		t.Fatal("Exchange should fail on temporarily_unavailable")
	}

	oboErr, ok := err.(*OBOError)
	if !ok {
		t.Fatalf("expected *OBOError, got %T", err)
	}
	if !oboErr.Retryable {
		t.Error("temporarily_unavailable should be retryable")
	}
}

func TestOBOError_Interface(t *testing.T) {
	err := &OBOError{
		Code:        "invalid_grant",
		Description: "token expired",
		Retryable:   false,
		Guidance:    "user must sign in again",
	}

	if err.Error() == "" {
		t.Error("Error() should return non-empty string")
	}
	if err.IsRetryable() {
		t.Error("invalid_grant should not be retryable")
	}
}

func TestClassifyOBOError_UnknownCode(t *testing.T) {
	err := classifyOBOError(400, "unknown_error", "something went wrong")
	if err == nil {
		t.Fatal("classifyOBOError should return non-nil error")
	}
	// Unknown errors should not be OBOError type.
	if _, ok := err.(*OBOError); ok {
		t.Error("unknown errors should not be *OBOError")
	}
}


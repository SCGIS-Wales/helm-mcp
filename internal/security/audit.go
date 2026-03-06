package security

import (
	"log/slog"
	"time"
)

// AuditEvent represents a security-relevant event for audit logging.
//
// Each application in the chain must log:
//   - Timestamp (UTC)
//   - Caller app ID (azp claim)
//   - Caller user OID/UPN
//   - Caller tenant ID
//   - Received scopes (scp claim)
//   - Resource accessed / action performed
//   - Result status
//   - OBO token source (cache/provider)
//   - OBO latency
type AuditEvent struct {
	// Timestamp is the UTC time of the event.
	Timestamp time.Time `json:"timestamp"`
	// EventType classifies the event (auth_success, auth_failure, authz_denied, obo_exchange, tool_call).
	EventType string `json:"event_type"`
	// PrincipalID is the user's object ID (oid claim).
	PrincipalID string `json:"principal_id,omitempty"`
	// PrincipalName is the user's UPN or preferred_username.
	PrincipalName string `json:"principal_name,omitempty"`
	// TenantID is the Entra ID tenant.
	TenantID string `json:"tenant_id,omitempty"`
	// ClientAppID is the calling application (azp/appid claim).
	ClientAppID string `json:"client_app_id,omitempty"`
	// Scopes are the token's delegated scopes.
	Scopes []string `json:"scopes,omitempty"`
	// Roles are the token's app roles.
	Roles []string `json:"roles,omitempty"`
	// TokenID is the unique token identifier (jti/uti).
	TokenID string `json:"token_id,omitempty"`
	// SessionID is the MCP session identifier.
	SessionID string `json:"session_id,omitempty"`
	// Action describes the operation performed (e.g., "helm_install", "helm_list").
	Action string `json:"action,omitempty"`
	// Resource describes the target resource (e.g., namespace/release).
	Resource string `json:"resource,omitempty"`
	// Result is the outcome (success, denied, error).
	Result string `json:"result"`
	// Error contains the error message if Result is not "success".
	Error string `json:"error,omitempty"`
	// DurationMS is the operation duration in milliseconds.
	DurationMS int64 `json:"duration_ms,omitempty"`
	// OBOTarget is the downstream API for OBO exchanges.
	OBOTarget string `json:"obo_target,omitempty"`
	// OBOTokenSource indicates whether the OBO token came from cache or provider.
	OBOTokenSource string `json:"obo_token_source,omitempty"`
	// RemoteAddr is the client's IP address.
	RemoteAddr string `json:"remote_addr,omitempty"`
}

// AuditLogger emits structured security audit events via slog.
// All events are logged at INFO level to ensure they appear in production logs.
type AuditLogger struct {
	logger *slog.Logger
}

// NewAuditLogger creates a new audit logger. If logger is nil, slog.Default() is used.
func NewAuditLogger(logger *slog.Logger) *AuditLogger {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuditLogger{logger: logger}
}

// Log emits an audit event as a structured log entry.
func (a *AuditLogger) Log(event AuditEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	attrs := []slog.Attr{
		slog.String("audit.event_type", event.EventType),
		slog.String("audit.result", event.Result),
		slog.Time("audit.timestamp", event.Timestamp),
	}

	if event.PrincipalID != "" {
		attrs = append(attrs, slog.String("audit.principal_id", event.PrincipalID))
	}
	if event.PrincipalName != "" {
		attrs = append(attrs, slog.String("audit.principal_name", event.PrincipalName))
	}
	if event.TenantID != "" {
		attrs = append(attrs, slog.String("audit.tenant_id", event.TenantID))
	}
	if event.ClientAppID != "" {
		attrs = append(attrs, slog.String("audit.client_app_id", event.ClientAppID))
	}
	if event.TokenID != "" {
		attrs = append(attrs, slog.String("audit.token_id", event.TokenID))
	}
	if event.SessionID != "" {
		attrs = append(attrs, slog.String("audit.session_id", event.SessionID))
	}
	if event.Action != "" {
		attrs = append(attrs, slog.String("audit.action", event.Action))
	}
	if event.Resource != "" {
		attrs = append(attrs, slog.String("audit.resource", event.Resource))
	}
	if event.Error != "" {
		attrs = append(attrs, slog.String("audit.error", event.Error))
	}
	if event.DurationMS > 0 {
		attrs = append(attrs, slog.Int64("audit.duration_ms", event.DurationMS))
	}
	if event.OBOTarget != "" {
		attrs = append(attrs, slog.String("audit.obo_target", event.OBOTarget))
	}
	if event.OBOTokenSource != "" {
		attrs = append(attrs, slog.String("audit.obo_token_source", event.OBOTokenSource))
	}
	if event.RemoteAddr != "" {
		attrs = append(attrs, slog.String("audit.remote_addr", event.RemoteAddr))
	}
	if len(event.Scopes) > 0 {
		attrs = append(attrs, slog.String("audit.scopes", joinScopes(event.Scopes)))
	}

	// Convert []slog.Attr to []any for LogAttrs.
	a.logger.LogAttrs(nil, slog.LevelInfo, "security_audit", attrs...) //nolint:staticcheck // nil context is intentional for audit log consistency
}

// LogAuthSuccess logs a successful authentication event.
func (a *AuditLogger) LogAuthSuccess(claims *TokenClaims, sessionID, remoteAddr string) {
	a.Log(AuditEvent{
		EventType:     "auth_success",
		PrincipalID:   claims.ObjectID,
		PrincipalName: claims.PreferredUsername,
		TenantID:      claims.TenantID,
		ClientAppID:   claims.AuthorizedParty,
		Scopes:        claims.Scopes,
		Roles:         claims.Roles,
		TokenID:       claims.TokenID,
		SessionID:     sessionID,
		Result:        "success",
		RemoteAddr:    remoteAddr,
	})
}

// LogAuthFailure logs a failed authentication attempt.
func (a *AuditLogger) LogAuthFailure(reason, remoteAddr string) {
	a.Log(AuditEvent{
		EventType:  "auth_failure",
		Result:     "denied",
		Error:      reason,
		RemoteAddr: remoteAddr,
	})
}

// LogAuthzDenied logs an authorization denial.
func (a *AuditLogger) LogAuthzDenied(claims *TokenClaims, action, reason string) {
	a.Log(AuditEvent{
		EventType:     "authz_denied",
		PrincipalID:   claims.ObjectID,
		PrincipalName: claims.PreferredUsername,
		TenantID:      claims.TenantID,
		ClientAppID:   claims.AuthorizedParty,
		Scopes:        claims.Scopes,
		Action:        action,
		Result:        "denied",
		Error:         reason,
	})
}

// LogOBOExchange logs an OBO token exchange event.
func (a *AuditLogger) LogOBOExchange(claims *TokenClaims, target, tokenSource string, durationMS int64, err error) {
	event := AuditEvent{
		EventType:      "obo_exchange",
		PrincipalID:    claims.ObjectID,
		PrincipalName:  claims.PreferredUsername,
		TenantID:       claims.TenantID,
		ClientAppID:    claims.AuthorizedParty,
		OBOTarget:      target,
		OBOTokenSource: tokenSource,
		DurationMS:     durationMS,
		Result:         "success",
	}
	if err != nil {
		event.Result = "error"
		event.Error = err.Error()
	}
	a.Log(event)
}

func joinScopes(scopes []string) string {
	result := ""
	for i, s := range scopes {
		if i > 0 {
			result += " "
		}
		result += s
	}
	return result
}

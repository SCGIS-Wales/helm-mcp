package security

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func newTestAuditLogger(buf *bytes.Buffer) *AuditLogger {
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)
	return NewAuditLogger(logger)
}

func TestAuditLogger_Log(t *testing.T) {
	var buf bytes.Buffer
	al := newTestAuditLogger(&buf)

	al.Log(AuditEvent{
		EventType:   "test_event",
		PrincipalID: "user-123",
		TenantID:    "tenant-abc",
		ClientAppID: "client-1",
		Action:      "helm_install",
		Resource:    "default/nginx",
		Result:      "success",
		DurationMS:  42,
	})

	output := buf.String()

	expectations := []string{
		"security_audit",
		"audit.event_type=test_event",
		"audit.result=success",
		"audit.principal_id=user-123",
		"audit.tenant_id=tenant-abc",
		"audit.client_app_id=client-1",
		"audit.action=helm_install",
		"audit.resource=default/nginx",
		"audit.duration_ms=42",
	}

	for _, exp := range expectations {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in audit log, got:\n%s", exp, output)
		}
	}
}

func TestAuditLogger_LogAuthSuccess(t *testing.T) {
	var buf bytes.Buffer
	al := newTestAuditLogger(&buf)

	claims := &TokenClaims{
		ObjectID:          "oid-123",
		PreferredUsername:  "user@example.com",
		TenantID:          "tenant-1",
		AuthorizedParty:   "client-app-1",
		Scopes:            []string{"helm.read", "helm.write"},
		Roles:             []string{"HelmOperator"},
		TokenID:           "token-xyz",
	}

	al.LogAuthSuccess(claims, "session-1", "10.0.0.1:54321")

	output := buf.String()
	if !strings.Contains(output, "auth_success") {
		t.Error("expected auth_success event type")
	}
	if !strings.Contains(output, "audit.principal_id=oid-123") {
		t.Error("expected principal_id in log")
	}
	if !strings.Contains(output, "audit.remote_addr=10.0.0.1:54321") {
		t.Error("expected remote_addr in log")
	}
}

func TestAuditLogger_LogAuthFailure(t *testing.T) {
	var buf bytes.Buffer
	al := newTestAuditLogger(&buf)

	al.LogAuthFailure("token expired", "10.0.0.1:54321")

	output := buf.String()
	if !strings.Contains(output, "auth_failure") {
		t.Error("expected auth_failure event type")
	}
	if !strings.Contains(output, "audit.error=\"token expired\"") {
		t.Error("expected error reason in log")
	}
	if !strings.Contains(output, "audit.result=denied") {
		t.Error("expected denied result")
	}
}

func TestAuditLogger_LogAuthzDenied(t *testing.T) {
	var buf bytes.Buffer
	al := newTestAuditLogger(&buf)

	claims := &TokenClaims{
		ObjectID:         "oid-123",
		PreferredUsername: "user@example.com",
		TenantID:         "tenant-1",
		AuthorizedParty:  "client-1",
		Scopes:           []string{"helm.read"},
	}

	al.LogAuthzDenied(claims, "helm_install", "missing required scope: helm.write")

	output := buf.String()
	if !strings.Contains(output, "authz_denied") {
		t.Error("expected authz_denied event type")
	}
	if !strings.Contains(output, "audit.action=helm_install") {
		t.Error("expected action in log")
	}
}

func TestAuditLogger_LogOBOExchange_Success(t *testing.T) {
	var buf bytes.Buffer
	al := newTestAuditLogger(&buf)

	claims := &TokenClaims{
		ObjectID:         "oid-123",
		PreferredUsername: "user@example.com",
		TenantID:         "tenant-1",
		AuthorizedParty:  "client-1",
	}

	al.LogOBOExchange(claims, "api://downstream", "provider", 150, nil)

	output := buf.String()
	if !strings.Contains(output, "obo_exchange") {
		t.Error("expected obo_exchange event type")
	}
	if !strings.Contains(output, "audit.obo_target=api://downstream") {
		t.Error("expected obo_target in log")
	}
	if !strings.Contains(output, "audit.obo_token_source=provider") {
		t.Error("expected obo_token_source in log")
	}
	if !strings.Contains(output, "audit.result=success") {
		t.Error("expected success result")
	}
}

func TestAuditLogger_LogOBOExchange_Error(t *testing.T) {
	var buf bytes.Buffer
	al := newTestAuditLogger(&buf)

	claims := &TokenClaims{
		ObjectID: "oid-123",
	}

	al.LogOBOExchange(claims, "api://downstream", "provider", 50,
		&testError{"OBO exchange failed"})

	output := buf.String()
	if !strings.Contains(output, "audit.result=error") {
		t.Error("expected error result")
	}
}

func TestAuditLogger_NilLogger(t *testing.T) {
	// NewAuditLogger with nil should use slog.Default() and not panic.
	al := NewAuditLogger(nil)
	al.Log(AuditEvent{
		EventType: "test",
		Result:    "success",
	})
	// Just verifying it doesn't panic.
}

func TestAuditLogger_TimestampAutoSet(t *testing.T) {
	var buf bytes.Buffer
	al := newTestAuditLogger(&buf)

	before := time.Now().UTC()
	al.Log(AuditEvent{
		EventType: "test",
		Result:    "success",
	})
	// Timestamp should have been auto-set.
	output := buf.String()
	if !strings.Contains(output, "audit.timestamp=") {
		t.Error("expected auto-set timestamp")
	}
	_ = before // Verified by log presence.
}

func TestJoinScopes(t *testing.T) {
	tests := []struct {
		scopes []string
		want   string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b", "c"}, "a b c"},
	}
	for _, tt := range tests {
		got := joinScopes(tt.scopes)
		if got != tt.want {
			t.Errorf("joinScopes(%v) = %q, want %q", tt.scopes, got, tt.want)
		}
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

package resilience

import (
	"strings"
	"testing"
)

func TestTruncateResponse_NoTruncation(t *testing.T) {
	response := "short response"
	result := TruncateResponse(response, DefaultMaxResponseBytes)
	if result != response {
		t.Errorf("expected no truncation, got %q", result)
	}
}

func TestTruncateResponse_DisabledWithZero(t *testing.T) {
	response := "any response"
	result := TruncateResponse(response, 0)
	if result != response {
		t.Error("expected truncation disabled with maxBytes=0")
	}
}

func TestTruncateResponse_Truncated(t *testing.T) {
	response := strings.Repeat("x", 1000)
	result := TruncateResponse(response, 500)

	if len(result) > 500 {
		t.Errorf("expected result <= 500 bytes, got %d", len(result))
	}
	if !strings.Contains(result, "[Truncated:") {
		t.Error("expected truncation notice in result")
	}
	if !strings.Contains(result, "1000 bytes") {
		t.Error("expected original size in truncation notice")
	}
}

func TestTruncateResponse_PreservesNewlineBoundary(t *testing.T) {
	// Build response with newlines every 50 chars.
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		sb.WriteString(strings.Repeat("a", 49))
		sb.WriteByte('\n')
	}
	response := sb.String() // 1000 chars

	result := TruncateResponse(response, 500)
	if !strings.Contains(result, "[Truncated:") {
		t.Error("expected truncation notice")
	}

	// The truncated content (before the notice) should end at a newline.
	parts := strings.SplitN(result, "\n\n[Truncated:", 2)
	if len(parts) < 2 {
		t.Fatal("expected truncation notice separator")
	}
	content := parts[0]
	if content[len(content)-1] != '\n' {
		t.Error("expected truncation at newline boundary")
	}
}

func TestTruncateResponse_ExactBoundary(t *testing.T) {
	response := strings.Repeat("x", 256)
	result := TruncateResponse(response, 256)
	if result != response {
		t.Error("expected no truncation when response equals maxBytes")
	}
}

func TestTruncateResponse_NegativeMaxBytes(t *testing.T) {
	response := "test"
	result := TruncateResponse(response, -1)
	if result != response {
		t.Error("expected no truncation with negative maxBytes")
	}
}

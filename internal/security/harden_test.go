package security

import (
	"runtime"
	"testing"
)

func TestApplyHardening_ReturnsResult(t *testing.T) {
	result := ApplyHardening(HardenOptions{})
	if result.Platform == "" {
		t.Error("expected non-empty Platform in HardenResult")
	}
	if result.Platform != runtime.GOOS {
		t.Errorf("Platform = %q, want %q", result.Platform, runtime.GOOS)
	}
}

func TestApplyHardening_DisableAll(t *testing.T) {
	result := ApplyHardening(HardenOptions{DisableAll: true})
	if result.Skipped == "" {
		t.Error("expected non-empty Skipped when DisableAll is true")
	}
	if result.Dumpable {
		t.Error("expected Dumpable=false when hardening is disabled")
	}
	if result.CapabilitiesDropped {
		t.Error("expected CapabilitiesDropped=false when hardening is disabled")
	}
}

func TestApplyHardening_DisableAll_Debug(t *testing.T) {
	// Exercise the debug logging path with DisableAll.
	result := ApplyHardening(HardenOptions{DisableAll: true, Debug: true})
	if result.Skipped == "" {
		t.Error("expected non-empty Skipped when DisableAll is true")
	}
}

func TestHardenResult_String(t *testing.T) {
	// Skipped case
	r := HardenResult{Platform: "linux", Skipped: "disabled"}
	s := r.String()
	if s == "" {
		t.Error("expected non-empty string from HardenResult.String()")
	}

	// Non-skipped case
	r = HardenResult{Platform: "linux", Dumpable: true, CapabilitiesDropped: true, CapabilitiesCount: 5}
	s = r.String()
	if s == "" {
		t.Error("expected non-empty string from HardenResult.String()")
	}
}

func TestHardenResult_String_WithErrors(t *testing.T) {
	r := HardenResult{
		Platform: "linux",
		Errors:   []string{"error1", "error2"},
	}
	s := r.String()
	if s == "" {
		t.Error("expected non-empty string from HardenResult.String()")
	}
}

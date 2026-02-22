package security

import (
	"fmt"
	"log"
)

const securityLogPrefix = "[security] "

// secLog logs a message with the security prefix when debug is enabled.
func secLog(debug bool, format string, args ...any) {
	if debug {
		log.Printf(securityLogPrefix+format, args...)
	}
}

// HardenOptions controls which process hardening mechanisms to apply.
type HardenOptions struct {
	// DisableAll skips all hardening (for debugging with --no-harden).
	DisableAll bool
	// Debug enables logging of applied hardening measures to stderr.
	Debug bool
}

// HardenResult reports which hardening mechanisms were applied.
type HardenResult struct {
	// Dumpable is true if PR_SET_DUMPABLE was set to 0 (Linux only).
	// This blocks ptrace attach, core dumps, and /proc/pid/mem reads.
	Dumpable bool
	// CapabilitiesDropped is true if capabilities were dropped from the
	// bounding set (Linux only).
	CapabilitiesDropped bool
	// CapabilitiesCount is the number of capabilities that were dropped.
	CapabilitiesCount int
	// Platform is the runtime platform (e.g. "linux", "darwin").
	Platform string
	// Skipped is non-empty if hardening was skipped, with the reason.
	Skipped string
	// Errors collects non-fatal errors encountered during hardening.
	Errors []string
}

// String returns a human-readable summary of the hardening result.
func (r HardenResult) String() string {
	if r.Skipped != "" {
		return fmt.Sprintf("hardening skipped on %s: %s", r.Platform, r.Skipped)
	}
	return fmt.Sprintf(
		"platform=%s dumpable=%v caps_dropped=%v(%d) errors=%d",
		r.Platform, r.Dumpable, r.CapabilitiesDropped, r.CapabilitiesCount, len(r.Errors),
	)
}

// ApplyHardening applies platform-specific process hardening.
// On non-Linux platforms, this is a no-op that returns a result
// indicating hardening was skipped.
//
// This function should be called as early as possible in main(),
// before any sensitive data is loaded. Hardening failures are
// non-fatal: they are recorded in HardenResult.Errors but do not
// prevent the server from starting.
func ApplyHardening(opts HardenOptions) HardenResult {
	return applyHardening(opts)
}

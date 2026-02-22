//go:build !linux

package security

import (
	"fmt"
	"runtime"
)

func applyHardening(opts HardenOptions) HardenResult {
	result := HardenResult{
		Platform: runtime.GOOS,
		Skipped:  fmt.Sprintf("process hardening not available on %s (Linux-only)", runtime.GOOS),
	}
	secLog(opts.Debug, "%s", result.Skipped)
	return result
}

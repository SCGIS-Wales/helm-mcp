//go:build linux

package security

import (
	"fmt"
	"runtime"
	"syscall"
)

// Linux prctl constants.
const (
	prSetDumpable = 4  // PR_SET_DUMPABLE
	prGetDumpable = 3  // PR_GET_DUMPABLE
	prCapBsetDrop = 24 // PR_CAPBSET_DROP
	prCapBsetRead = 23 // PR_CAPBSET_READ

	// defaultLastCap is a safe upper bound for the highest capability
	// number. The actual value is read from /proc/sys/kernel/cap_last_cap.
	defaultLastCap = 40
)

func applyHardening(opts HardenOptions) HardenResult {
	result := HardenResult{Platform: runtime.GOOS}

	if opts.DisableAll {
		result.Skipped = "hardening disabled via --no-harden"
		secLog(opts.Debug, "%s", result.Skipped)
		return result
	}

	setDumpable(&result, opts.Debug)
	dropCapabilities(&result, opts.Debug)

	return result
}

// setDumpable sets PR_SET_DUMPABLE to 0, blocking ptrace attach,
// core dumps, and /proc/pid/mem reads from non-root processes.
func setDumpable(result *HardenResult, debug bool) {
	_, _, errno := syscall.RawSyscall(
		syscall.SYS_PRCTL,
		prSetDumpable,
		0, 0,
	)
	if errno != 0 {
		errMsg := fmt.Sprintf("PR_SET_DUMPABLE failed: %v", errno)
		result.Errors = append(result.Errors, errMsg)
		secLog(debug, "%s", errMsg)
		return
	}

	// Verify the setting was applied by reading it back.
	ret, _, errno := syscall.RawSyscall(
		syscall.SYS_PRCTL,
		prGetDumpable,
		0, 0,
	)
	if errno != 0 {
		errMsg := fmt.Sprintf("PR_GET_DUMPABLE verification failed: %v", errno)
		result.Errors = append(result.Errors, errMsg)
		secLog(debug, "%s", errMsg)
		return
	}

	if ret == 0 {
		result.Dumpable = true
		secLog(debug, "PR_SET_DUMPABLE set to 0 (ptrace/coredump blocked)")
	} else {
		errMsg := fmt.Sprintf("PR_SET_DUMPABLE verification: expected 0, got %d", ret)
		result.Errors = append(result.Errors, errMsg)
		secLog(debug, "%s", errMsg)
	}
}

// dropCapabilities drops all capabilities from the bounding set.
// For non-root processes this is typically a no-op (capabilities are
// already absent), but it protects against container misconfigurations
// that grant unnecessary capabilities.
func dropCapabilities(result *HardenResult, debug bool) {
	lastCap := readLastCap()
	dropped := 0
	skipped := 0

	for cap := uintptr(0); cap <= lastCap; cap++ {
		// Check if capability is in the bounding set.
		ret, _, errno := syscall.RawSyscall(
			syscall.SYS_PRCTL,
			prCapBsetRead,
			cap, 0,
		)
		if errno != 0 {
			// Capability number may not exist on this kernel; skip.
			skipped++
			continue
		}
		if ret != 1 {
			// Capability not in bounding set; nothing to drop.
			continue
		}

		// Drop it.
		_, _, errno = syscall.RawSyscall(
			syscall.SYS_PRCTL,
			prCapBsetDrop,
			cap, 0,
		)
		if errno != 0 {
			errMsg := fmt.Sprintf("failed to drop capability %d: %v", cap, errno)
			result.Errors = append(result.Errors, errMsg)
			secLog(debug, "%s", errMsg)
		} else {
			dropped++
		}
	}

	result.CapabilitiesDropped = dropped > 0
	result.CapabilitiesCount = dropped

	if dropped > 0 {
		secLog(debug, "dropped %d capabilities from bounding set (skipped %d)", dropped, skipped)
	} else {
		secLog(debug, "no capabilities to drop (already running unprivileged, skipped %d)", skipped)
	}
}

// readLastCap reads /proc/sys/kernel/cap_last_cap to determine the
// highest valid capability number on this kernel. Falls back to
// defaultLastCap if the file cannot be read.
func readLastCap() uintptr {
	data := make([]byte, 16)
	fd, err := syscall.Open("/proc/sys/kernel/cap_last_cap", syscall.O_RDONLY, 0)
	if err != nil {
		return defaultLastCap
	}
	defer syscall.Close(fd) //nolint:errcheck

	n, err := syscall.Read(fd, data)
	if err != nil || n == 0 {
		return defaultLastCap
	}

	var val uintptr
	for i := 0; i < n; i++ {
		if data[i] >= '0' && data[i] <= '9' {
			val = val*10 + uintptr(data[i]-'0')
		} else {
			break // stop at newline or any non-digit
		}
	}
	if val > 0 {
		return val
	}
	return defaultLastCap
}

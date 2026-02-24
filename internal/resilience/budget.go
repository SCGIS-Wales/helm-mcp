package resilience

import (
	"fmt"
)

// DefaultMaxResponseBytes is the default maximum response size (256 KB).
const DefaultMaxResponseBytes = 256 * 1024

// TruncateResponse truncates a response string if it exceeds maxBytes.
// When truncated, it appends metadata indicating the original and returned sizes.
// A maxBytes of 0 disables truncation.
func TruncateResponse(response string, maxBytes int) string {
	if maxBytes <= 0 || len(response) <= maxBytes {
		return response
	}

	// Reserve space for the truncation notice (~120 bytes).
	const reserveBytes = 150
	cutoff := maxBytes - reserveBytes
	if cutoff < 0 {
		cutoff = 0
	}

	// Cut at a newline boundary if possible to avoid breaking mid-line.
	truncated := response[:cutoff]
	for i := len(truncated) - 1; i > cutoff-200 && i >= 0; i-- {
		if truncated[i] == '\n' {
			truncated = truncated[:i+1]
			break
		}
	}

	notice := fmt.Sprintf(
		"\n\n[Truncated: response was %d bytes, showing first %d bytes. "+
			"Use more specific queries or filters to reduce output size.]",
		len(response), len(truncated),
	)
	return truncated + notice
}

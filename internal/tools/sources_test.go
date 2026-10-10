package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValueSetInput_ValidateFollowsHelmParsing(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "ok.txt")
	if err := os.WriteFile(ok, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		entry string
		want  string // empty means valid
	}{
		{"plain path", "a=" + ok, ""},
		{"list of valid paths", "a={" + ok + "," + ok + "}", ""},
		{"sensitive path", "a=/etc/passwd", "sensitive"},
		// Helm reads {a,b} as a list of files; validating the raw remainder
		// as one path let these through.
		{"sensitive path inside a list", "a={/etc/passwd}", "sensitive"},
		{"second pair after a comma", "a=" + ok + ",b=/etc/shadow", "sensitive"},
		{"traversal", "a=../secret", ".."},
		{"URL", "a=https://example.com/x", "local files"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (&ValueSetInput{SetFile: []string{tt.entry}}).Validate()
			if tt.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
		})
	}
}

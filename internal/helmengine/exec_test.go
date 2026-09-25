package helmengine

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestHelmCommandSetsWaitDelay(t *testing.T) {
	cmd := HelmCommand(context.Background(), "version")
	if cmd.WaitDelay != CommandWaitDelay {
		t.Fatalf("WaitDelay = %v, want %v", cmd.WaitDelay, CommandWaitDelay)
	}
	if len(cmd.Args) != 2 || cmd.Args[1] != "version" {
		t.Fatalf("unexpected args %v", cmd.Args)
	}
}

// A child that outlives the killed parent keeps the output pipe open. Without
// WaitDelay, Run blocks until that child exits instead of honouring ctx.
func TestNewCommandReturnsWhenGrandchildHoldsPipe(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires a POSIX shell")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	cmd := newCommand(ctx, "sh", "-c", "sleep 60 & wait")
	cmd.WaitDelay = 500 * time.Millisecond
	var out stringsBuilder
	cmd.Stdout = &out

	start := time.Now()
	if err := cmd.Run(); err == nil {
		t.Fatal("expected an error from the cancelled command")
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("Run returned after %v; WaitDelay was not honoured", elapsed)
	}
}

type stringsBuilder struct{ b []byte }

func (s *stringsBuilder) Write(p []byte) (int, error) { s.b = append(s.b, p...); return len(p), nil }

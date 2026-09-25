package helmengine

import (
	"context"
	"os/exec"
	"time"
)

// CommandWaitDelay bounds how long a helm CLI invocation may linger after its
// context is cancelled. Without it, exec.Cmd.Wait blocks until every process
// holding the stdout/stderr pipes exits: a plugin install hook that spawns a
// child (git, curl, a downloaded binary) kept the call hanging long after the
// timeout had killed helm itself.
const CommandWaitDelay = 10 * time.Second

// HelmCommand returns an exec.Cmd running the helm CLI with args, bound to
// ctx and with WaitDelay set so the timeout is actually honoured.
func HelmCommand(ctx context.Context, args ...string) *exec.Cmd {
	return newCommand(ctx, "helm", args...)
}

func newCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = CommandWaitDelay
	return cmd
}

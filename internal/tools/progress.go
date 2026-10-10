package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ProgressInterval is how often WithProgress sends a heartbeat. A variable
// so tests can shorten it.
var ProgressInterval = 10 * time.Second

// WithProgress runs fn and, when the client asked for progress by sending a
// progress token, sends a notifications/progress heartbeat every
// ProgressInterval until fn returns.
//
// Helm reports no intermediate progress, so each notification carries an
// increasing counter and the elapsed time. Its purpose is practical: clients
// that reset their request timeout on progress keep waiting for a long
// install or upgrade with wait set, instead of abandoning it.
func WithProgress[T any](ctx context.Context, req *mcp.CallToolRequest, operation string, fn func() (T, error)) (T, error) {
	token := progressToken(req)
	if token == nil {
		return fn()
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(ProgressInterval)
		defer ticker.Stop()
		start := time.Now()
		for n := 1; ; n++ {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				// A failed notification must not fail the operation.
				_ = req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
					ProgressToken: token,
					Progress:      float64(n),
					Message:       fmt.Sprintf("%s still running after %s", operation, time.Since(start).Round(time.Second)),
				})
			}
		}
	}()
	defer func() {
		close(done)
		wg.Wait()
	}()

	return fn()
}

func progressToken(req *mcp.CallToolRequest) any {
	if req == nil || req.Session == nil || req.Params == nil {
		return nil
	}
	return req.Params.GetProgressToken()
}

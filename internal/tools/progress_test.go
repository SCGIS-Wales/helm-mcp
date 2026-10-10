package tools

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type slowInput struct {
	Wait string `json:"wait" jsonschema:"how long to run"`
}

// connectSlowTool serves one tool that runs for the requested duration
// inside WithProgress, and returns a client that counts progress
// notifications.
func connectSlowTool(t *testing.T, received *atomic.Int32) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "slow"}, func(ctx context.Context, req *mcp.CallToolRequest, in slowInput) (*mcp.CallToolResult, any, error) {
		d, _ := time.ParseDuration(in.Wait)
		out, err := WithProgress(ctx, req, "slow", func() (string, error) {
			time.Sleep(d)
			return "done", nil
		})
		return TextResult(out), nil, err
	})

	st, ct := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "0"}, &mcp.ClientOptions{
		ProgressNotificationHandler: func(context.Context, *mcp.ProgressNotificationClientRequest) {
			received.Add(1)
		},
	})
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestWithProgress_SendsHeartbeats(t *testing.T) {
	orig := ProgressInterval
	ProgressInterval = 20 * time.Millisecond
	t.Cleanup(func() { ProgressInterval = orig })

	var received atomic.Int32
	cs := connectSlowTool(t, &received)

	params := &mcp.CallToolParams{Name: "slow", Arguments: map[string]any{"wait": "150ms"}}
	params.SetProgressToken("tok")
	res, err := cs.CallTool(context.Background(), params)
	if err != nil || res.IsError {
		t.Fatalf("call failed: %v %+v", err, res)
	}

	deadline := time.Now().Add(time.Second)
	for received.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if received.Load() < 2 {
		t.Errorf("expected heartbeats while the tool ran, got %d", received.Load())
	}
}

func TestWithProgress_SilentWithoutToken(t *testing.T) {
	orig := ProgressInterval
	ProgressInterval = 10 * time.Millisecond
	t.Cleanup(func() { ProgressInterval = orig })

	var received atomic.Int32
	cs := connectSlowTool(t, &received)

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "slow", Arguments: map[string]any{"wait": "60ms"}})
	if err != nil || res.IsError {
		t.Fatalf("call failed: %v %+v", err, res)
	}
	time.Sleep(30 * time.Millisecond)
	if n := received.Load(); n != 0 {
		t.Errorf("no progress token means no notifications, got %d", n)
	}
}

func TestWithProgress_NilRequest(t *testing.T) {
	got, err := WithProgress(context.Background(), nil, "op", func() (int, error) { return 7, nil })
	if err != nil || got != 7 {
		t.Errorf("a nil request should just run fn, got %d, %v", got, err)
	}
}

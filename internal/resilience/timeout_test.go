package resilience

import (
	"context"
	"testing"
	"time"
)

func TestDefaultTimeout(t *testing.T) {
	tests := []struct {
		category ToolCategory
		want     time.Duration
	}{
		{CategoryQuery, 30 * time.Second},
		{CategoryMutate, 120 * time.Second},
		{CategoryChart, 60 * time.Second},
		{CategoryRepo, 60 * time.Second},
		{ToolCategory(99), 30 * time.Second},
	}
	for _, tt := range tests {
		if got := DefaultTimeout(tt.category); got != tt.want {
			t.Errorf("DefaultTimeout(%d) = %v, want %v", tt.category, got, tt.want)
		}
	}
}

func TestWithToolTimeout_SetsDeadline(t *testing.T) {
	ctx := context.Background()
	tctx, cancel := WithToolTimeout(ctx, CategoryQuery)
	defer cancel()

	deadline, ok := tctx.Deadline()
	if !ok {
		t.Fatal("expected deadline to be set")
	}

	remaining := time.Until(deadline)
	if remaining < 25*time.Second || remaining > 31*time.Second {
		t.Errorf("expected ~30s remaining, got %v", remaining)
	}
}

func TestWithToolTimeout_RespectsExistingShorterDeadline(t *testing.T) {
	ctx, parentCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer parentCancel()

	tctx, cancel := WithToolTimeout(ctx, CategoryMutate)
	defer cancel()

	deadline, ok := tctx.Deadline()
	if !ok {
		t.Fatal("expected deadline to be set")
	}

	remaining := time.Until(deadline)
	// Should respect the parent's 5s deadline, not the category's 120s.
	if remaining > 6*time.Second {
		t.Errorf("expected <= 5s remaining (parent deadline), got %v", remaining)
	}
}

func TestWithToolTimeout_OverridesLongerParent(t *testing.T) {
	ctx, parentCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer parentCancel()

	tctx, cancel := WithToolTimeout(ctx, CategoryQuery)
	defer cancel()

	deadline, ok := tctx.Deadline()
	if !ok {
		t.Fatal("expected deadline to be set")
	}

	remaining := time.Until(deadline)
	// Should use the category's 30s timeout, not the parent's 5 minutes.
	if remaining > 31*time.Second {
		t.Errorf("expected <= 30s remaining (category timeout), got %v", remaining)
	}
}

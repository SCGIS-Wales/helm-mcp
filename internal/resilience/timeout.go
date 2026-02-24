package resilience

import (
	"context"
	"time"
)

// ToolCategory classifies tools by their expected latency profile.
type ToolCategory int

const (
	// CategoryQuery covers read-only, fast operations (list, status, get, search, env).
	CategoryQuery ToolCategory = iota
	// CategoryMutate covers write operations (install, upgrade, uninstall, rollback, test).
	CategoryMutate
	// CategoryChart covers chart operations (template, lint, package, pull, push, show).
	CategoryChart
	// CategoryRepo covers repository management operations.
	CategoryRepo
)

// DefaultTimeout returns a sensible default timeout for a tool category.
func DefaultTimeout(category ToolCategory) time.Duration {
	switch category {
	case CategoryQuery:
		return 30 * time.Second
	case CategoryMutate:
		return 120 * time.Second
	case CategoryChart:
		return 60 * time.Second
	case CategoryRepo:
		return 60 * time.Second
	default:
		return 30 * time.Second
	}
}

// WithToolTimeout returns a context with a timeout appropriate for the tool category,
// unless the context already has a shorter deadline.
func WithToolTimeout(ctx context.Context, category ToolCategory) (context.Context, context.CancelFunc) {
	timeout := DefaultTimeout(category)

	// If the context already has a shorter deadline, respect it.
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			// Parent already has a tighter deadline; just wrap for cancel propagation.
			return context.WithCancel(ctx)
		}
	}

	return context.WithTimeout(ctx, timeout)
}

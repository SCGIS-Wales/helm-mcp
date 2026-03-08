package security

import (
	"log/slog"
	"sync"
	"time"
)

// SessionConfig configures the in-memory session/token cache.
//
// Design notes:
//   - Tokens are never used beyond their exp
//   - Preferred store: in-memory
//   - Cache keys are SHA-256 hashes of the raw bearer token
type SessionConfig struct {
	// InactivityTTL is the maximum time a cached entry survives without access.
	// Default: 5 minutes. Configurable via HELM_MCP_SESSION_TTL environment variable.
	InactivityTTL time.Duration

	// CleanupInterval controls how often expired entries are purged. Default: 1 minute.
	CleanupInterval time.Duration

	// MaxEntries is the maximum number of cached entries. Default: 10000.
	MaxEntries int
}

// DefaultSessionConfig returns the default session configuration.
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		InactivityTTL:   5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		MaxEntries:      10000,
	}
}

// SessionEntry holds a cached validated token with access tracking.
type SessionEntry struct {
	Claims     *TokenClaims
	LastAccess time.Time
	CreatedAt  time.Time
}

// SessionCache provides an in-memory cache for validated OIDC tokens.
// Cache keys are SHA-256 hashes of the raw bearer token, which ensures
// consistent lookup/store and avoids holding raw tokens in memory.
type SessionCache struct {
	mu       sync.Mutex
	entries  map[string]*SessionEntry
	config   SessionConfig
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewSessionCache creates a new session cache with background cleanup.
func NewSessionCache(config SessionConfig) *SessionCache {
	if config.InactivityTTL <= 0 {
		config.InactivityTTL = 5 * time.Minute
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 1 * time.Minute
	}
	if config.MaxEntries <= 0 {
		config.MaxEntries = 10000
	}

	c := &SessionCache{
		entries: make(map[string]*SessionEntry),
		config:  config,
		stopCh:  make(chan struct{}),
	}

	go c.cleanupLoop()

	return c
}

// CacheKey generates a cache key from principal ID and session ID.
// Deprecated: The middleware now uses SHA-256 hashes of the raw token as cache keys.
// This function is retained for backward compatibility but is no longer used by the middleware.
func CacheKey(principalID, sessionID string) string {
	return principalID + ":" + sessionID
}

// Get retrieves a cached session entry if it exists, is not expired by
// inactivity, and the token has not passed its expiration time.
// Returns nil if the entry is not found or has expired.
func (c *SessionCache) Get(key string) *TokenClaims {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil
	}

	now := time.Now()

	// Never use tokens beyond their exp (hard requirement).
	if now.After(entry.Claims.ExpiresAt) {
		delete(c.entries, key)
		return nil
	}

	// Check inactivity TTL.
	if now.Sub(entry.LastAccess) > c.config.InactivityTTL {
		delete(c.entries, key)
		return nil
	}

	// Update last access time (sliding window).
	entry.LastAccess = now

	return entry.Claims
}

// Put stores validated token claims in the cache.
func (c *SessionCache) Put(key string, claims *TokenClaims) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Enforce max entries to prevent unbounded memory growth.
	if len(c.entries) >= c.config.MaxEntries {
		c.evictOldest()
	}

	now := time.Now()
	c.entries[key] = &SessionEntry{
		Claims:     claims,
		LastAccess: now,
		CreatedAt:  now,
	}
}

// Delete removes an entry from the cache.
func (c *SessionCache) Delete(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// Size returns the number of entries in the cache.
func (c *SessionCache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// Stop terminates the background cleanup goroutine. Safe to call multiple times.
func (c *SessionCache) Stop() {
	c.stopOnce.Do(func() { close(c.stopCh) })
}

// cleanupLoop periodically removes expired entries.
func (c *SessionCache) cleanupLoop() {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

// cleanup removes entries that have exceeded inactivity TTL or token expiry.
func (c *SessionCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expired := 0
	for key, entry := range c.entries {
		if now.After(entry.Claims.ExpiresAt) || now.Sub(entry.LastAccess) > c.config.InactivityTTL {
			delete(c.entries, key)
			expired++
		}
	}

	if expired > 0 {
		slog.Debug("session cache cleanup", "expired", expired, "remaining", len(c.entries))
	}
}

// evictOldest removes the entry with the oldest last access time.
// Must be called with mu held.
func (c *SessionCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccess
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

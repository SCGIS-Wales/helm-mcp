package security

import (
	"testing"
	"time"
)

// testCacheKey generates a cache key for tests (replaces removed CacheKey function).
func testCacheKey(principalID, sessionID string) string {
	return principalID + ":" + sessionID
}

func TestSessionCache_PutAndGet(t *testing.T) {
	cache := NewSessionCache(SessionConfig{
		InactivityTTL:   7 * time.Minute,
		CleanupInterval: 1 * time.Hour, // Long interval so cleanup doesn't interfere
		MaxEntries:      100,
	})
	defer cache.Stop()

	claims := &TokenClaims{
		Subject:   "user-123",
		ObjectID:  "oid-456",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	key := testCacheKey("oid-456", "session-1")
	cache.Put(key, claims)

	got := cache.Get(key)
	if got == nil {
		t.Fatal("expected cached claims, got nil")
	}
	if got.Subject != "user-123" {
		t.Errorf("Subject = %q, want %q", got.Subject, "user-123")
	}
}

func TestSessionCache_GetMissing(t *testing.T) {
	cache := NewSessionCache(DefaultSessionConfig())
	defer cache.Stop()

	got := cache.Get("nonexistent-key")
	if got != nil {
		t.Error("expected nil for missing key")
	}
}

func TestSessionCache_ExpiredToken(t *testing.T) {
	cache := NewSessionCache(SessionConfig{
		InactivityTTL:   7 * time.Minute,
		CleanupInterval: 1 * time.Hour,
		MaxEntries:      100,
	})
	defer cache.Stop()

	// Token already expired.
	claims := &TokenClaims{
		Subject:   "user-123",
		ObjectID:  "oid-456",
		ExpiresAt: time.Now().Add(-1 * time.Second),
	}

	key := testCacheKey("oid-456", "session-1")
	cache.Put(key, claims)

	got := cache.Get(key)
	if got != nil {
		t.Error("expected nil for expired token")
	}

	// Entry should have been removed.
	if cache.Size() != 0 {
		t.Errorf("expected 0 entries after expired token access, got %d", cache.Size())
	}
}

func TestSessionCache_InactivityTTL(t *testing.T) {
	cache := NewSessionCache(SessionConfig{
		InactivityTTL:   50 * time.Millisecond,
		CleanupInterval: 1 * time.Hour,
		MaxEntries:      100,
	})
	defer cache.Stop()

	claims := &TokenClaims{
		Subject:   "user-123",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	key := testCacheKey("user-123", "session-1")
	cache.Put(key, claims)

	// Should be present immediately.
	if got := cache.Get(key); got == nil {
		t.Fatal("expected cached claims immediately after put")
	}

	// Wait for inactivity TTL to expire.
	time.Sleep(60 * time.Millisecond)

	got := cache.Get(key)
	if got != nil {
		t.Error("expected nil after inactivity TTL")
	}
}

func TestSessionCache_SlidingWindow(t *testing.T) {
	cache := NewSessionCache(SessionConfig{
		InactivityTTL:   100 * time.Millisecond,
		CleanupInterval: 1 * time.Hour,
		MaxEntries:      100,
	})
	defer cache.Stop()

	claims := &TokenClaims{
		Subject:   "user-123",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	key := testCacheKey("user-123", "session-1")
	cache.Put(key, claims)

	// Access before TTL expires should reset the window.
	time.Sleep(50 * time.Millisecond)
	got := cache.Get(key) // This should reset LastAccess
	if got == nil {
		t.Fatal("expected cached claims before TTL")
	}

	// Wait another 50ms (total 100ms since last access but only 50ms since Get)
	time.Sleep(60 * time.Millisecond)
	got = cache.Get(key)
	if got == nil {
		t.Error("expected cached claims (sliding window should have extended TTL)")
	}
}

func TestSessionCache_Delete(t *testing.T) {
	cache := NewSessionCache(DefaultSessionConfig())
	defer cache.Stop()

	claims := &TokenClaims{
		Subject:   "user-123",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	key := testCacheKey("user-123", "session-1")
	cache.Put(key, claims)
	cache.Delete(key)

	if got := cache.Get(key); got != nil {
		t.Error("expected nil after delete")
	}
}

func TestSessionCache_MaxEntries(t *testing.T) {
	cache := NewSessionCache(SessionConfig{
		InactivityTTL:   7 * time.Minute,
		CleanupInterval: 1 * time.Hour,
		MaxEntries:      3,
	})
	defer cache.Stop()

	for i := 0; i < 5; i++ {
		claims := &TokenClaims{
			Subject:   "user",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		cache.Put(testCacheKey("user", string(rune('a'+i))), claims)
	}

	if size := cache.Size(); size > 3 {
		t.Errorf("cache size = %d, should not exceed MaxEntries=3", size)
	}
}

func TestSessionCache_Size(t *testing.T) {
	cache := NewSessionCache(DefaultSessionConfig())
	defer cache.Stop()

	if size := cache.Size(); size != 0 {
		t.Errorf("empty cache size = %d", size)
	}

	claims := &TokenClaims{
		Subject:   "user-123",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	cache.Put("key1", claims)
	cache.Put("key2", claims)

	if size := cache.Size(); size != 2 {
		t.Errorf("cache size = %d, want 2", size)
	}
}

func TestSessionCache_Cleanup(t *testing.T) {
	cache := NewSessionCache(SessionConfig{
		InactivityTTL:   50 * time.Millisecond,
		CleanupInterval: 30 * time.Millisecond,
		MaxEntries:      100,
	})
	defer cache.Stop()

	claims := &TokenClaims{
		Subject:   "user-123",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	cache.Put("key1", claims)

	// Wait for inactivity TTL + cleanup interval.
	time.Sleep(100 * time.Millisecond)

	if size := cache.Size(); size != 0 {
		t.Errorf("expected cleanup to remove expired entry, size = %d", size)
	}
}

func TestDefaultSessionConfig(t *testing.T) {
	config := DefaultSessionConfig()

	if config.InactivityTTL != 5*time.Minute {
		t.Errorf("InactivityTTL = %v, want 5m", config.InactivityTTL)
	}
	if config.CleanupInterval != 1*time.Minute {
		t.Errorf("CleanupInterval = %v, want 1m", config.CleanupInterval)
	}
	if config.MaxEntries != 10000 {
		t.Errorf("MaxEntries = %d, want 10000", config.MaxEntries)
	}
}

func TestSessionCache_CustomTTL(t *testing.T) {
	// Users can configure a custom TTL value.
	cache := NewSessionCache(SessionConfig{
		InactivityTTL:   30 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		MaxEntries:      100,
	})
	defer cache.Stop()

	if cache.config.InactivityTTL != 30*time.Minute {
		t.Errorf("InactivityTTL = %v, want 30m", cache.config.InactivityTTL)
	}
}

func TestSessionCache_NegativeTTL(t *testing.T) {
	cache := NewSessionCache(SessionConfig{
		InactivityTTL:   -1 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		MaxEntries:      100,
	})
	defer cache.Stop()

	if cache.config.InactivityTTL != 5*time.Minute {
		t.Errorf("InactivityTTL = %v, should default to 5m for negative values", cache.config.InactivityTTL)
	}
}

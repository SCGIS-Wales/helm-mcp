package security

import "runtime"

// SecureBytes wraps a byte slice containing sensitive data (tokens,
// passwords) and provides explicit zeroing. After calling Zero(),
// the underlying memory is overwritten with zeroes.
//
// Limitations: Go strings are immutable and GC-managed. SecureBytes
// reduces the credential lifetime in our code paths but cannot
// guarantee that all copies created by the runtime (string interning,
// GC relocation) are zeroed. This is a defense-in-depth measure,
// not a guarantee.
//
// Usage:
//
//	sb := NewSecureBytesFromString(password)
//	defer sb.Zero()
//	// use sb.String() or sb.Bytes() ...
type SecureBytes struct {
	data []byte
}

// NewSecureBytes creates a SecureBytes from an existing byte slice.
// The caller should avoid keeping other references to the slice.
func NewSecureBytes(data []byte) *SecureBytes {
	if data == nil {
		return &SecureBytes{}
	}
	return &SecureBytes{data: data}
}

// NewSecureBytesFromString copies a string into a zeroing-capable
// byte slice. The original string cannot be zeroed (Go strings are
// immutable), but this limits additional copies in credential paths.
func NewSecureBytesFromString(s string) *SecureBytes {
	if s == "" {
		return &SecureBytes{}
	}
	b := make([]byte, len(s))
	copy(b, s)
	return &SecureBytes{data: b}
}

// Bytes returns the underlying byte slice. The caller must not
// retain references after calling Zero().
func (s *SecureBytes) Bytes() []byte {
	if s == nil {
		return nil
	}
	return s.data
}

// String returns the content as a string.
func (s *SecureBytes) String() string {
	if s == nil || len(s.data) == 0 {
		return ""
	}
	return string(s.data)
}

// Zero overwrites the underlying byte slice with zeroes.
// Uses runtime.KeepAlive to prevent the compiler from eliding
// the zeroing loop as a dead store.
// Safe to call multiple times or on a nil receiver.
func (s *SecureBytes) Zero() {
	if s == nil || len(s.data) == 0 {
		return
	}
	for i := range s.data {
		s.data[i] = 0
	}
	runtime.KeepAlive(s.data)
	s.data = nil
}

// Len returns the length of the secure data, or 0 if nil/zeroed.
func (s *SecureBytes) Len() int {
	if s == nil {
		return 0
	}
	return len(s.data)
}

// IsZeroed returns true if Zero() has been called.
func (s *SecureBytes) IsZeroed() bool {
	if s == nil {
		return true
	}
	return s.data == nil
}

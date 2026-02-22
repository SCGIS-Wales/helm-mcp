package security

import (
	"testing"
)

func TestSecureBytes_NewAndString(t *testing.T) {
	sb := NewSecureBytes([]byte("hello"))
	if sb.String() != "hello" {
		t.Errorf("String() = %q, want %q", sb.String(), "hello")
	}
	if sb.Len() != 5 {
		t.Errorf("Len() = %d, want 5", sb.Len())
	}
}

func TestSecureBytes_FromString(t *testing.T) {
	sb := NewSecureBytesFromString("secret-token")
	if sb.String() != "secret-token" {
		t.Errorf("String() = %q, want %q", sb.String(), "secret-token")
	}
	if sb.Len() != 12 {
		t.Errorf("Len() = %d, want 12", sb.Len())
	}
}

func TestSecureBytes_Zero(t *testing.T) {
	original := []byte("sensitive-data")
	sb := NewSecureBytes(original)

	sb.Zero()

	// Verify all bytes are zero in the original backing array.
	for i, b := range original {
		if b != 0 {
			t.Errorf("byte[%d] = %d, want 0 after Zero()", i, b)
		}
	}

	// Verify the SecureBytes reports as zeroed.
	if !sb.IsZeroed() {
		t.Error("IsZeroed() should return true after Zero()")
	}
	if sb.Len() != 0 {
		t.Errorf("Len() = %d, want 0 after Zero()", sb.Len())
	}
	if sb.String() != "" {
		t.Errorf("String() = %q, want empty after Zero()", sb.String())
	}
}

func TestSecureBytes_FromString_Zero(t *testing.T) {
	sb := NewSecureBytesFromString("my-password")

	// Capture the bytes before zeroing.
	data := sb.Bytes()
	if len(data) != 11 {
		t.Fatalf("Bytes() length = %d, want 11", len(data))
	}

	sb.Zero()

	// Verify the copy was zeroed.
	for i, b := range data {
		if b != 0 {
			t.Errorf("byte[%d] = %d, want 0 after Zero()", i, b)
		}
	}
	if !sb.IsZeroed() {
		t.Error("IsZeroed() should return true after Zero()")
	}
}

func TestSecureBytes_EmptyZero(t *testing.T) {
	// Zero on empty data should not panic.
	sb := NewSecureBytes(nil)
	sb.Zero() // should not panic

	sb2 := NewSecureBytes([]byte{})
	sb2.Zero() // should not panic

	sb3 := NewSecureBytesFromString("")
	sb3.Zero() // should not panic
}

func TestSecureBytes_NilReceiver(t *testing.T) {
	var sb *SecureBytes

	// All methods should be safe on nil receiver.
	sb.Zero() // should not panic

	if sb.Bytes() != nil {
		t.Error("Bytes() on nil should return nil")
	}
	if sb.String() != "" {
		t.Error("String() on nil should return empty")
	}
	if sb.Len() != 0 {
		t.Error("Len() on nil should return 0")
	}
	if !sb.IsZeroed() {
		t.Error("IsZeroed() on nil should return true")
	}
}

func TestSecureBytes_DoubleZero(t *testing.T) {
	sb := NewSecureBytesFromString("test-value")
	sb.Zero()
	sb.Zero() // should not panic
	if !sb.IsZeroed() {
		t.Error("IsZeroed() should return true after double Zero()")
	}
}

func TestSecureBytes_BytesReturnsSameSlice(t *testing.T) {
	data := []byte("raw-data")
	sb := NewSecureBytes(data)

	// Bytes() should return the same slice (not a copy).
	got := sb.Bytes()
	if &got[0] != &data[0] {
		t.Error("Bytes() should return the same underlying array")
	}
}

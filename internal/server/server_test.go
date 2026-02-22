package server

import (
	"testing"
)

func TestNewServer(t *testing.T) {
	s := NewServer("")
	if s == nil {
		t.Fatal("NewServer() returned nil")
	}
}

func TestNewServerWithVersion(t *testing.T) {
	s := NewServer("1.2.3")
	if s == nil {
		t.Fatal("NewServer(version) returned nil")
	}
}

func TestServerConstants(t *testing.T) {
	if ServerName != "helm-mcp" {
		t.Errorf("ServerName = %q, want %q", ServerName, "helm-mcp")
	}
	if ServerVersion == "" {
		t.Error("ServerVersion should not be empty")
	}
}

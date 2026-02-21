package server

import (
	"testing"
)

func TestNewServer(t *testing.T) {
	s := NewServer()
	if s == nil {
		t.Fatal("NewServer() returned nil")
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

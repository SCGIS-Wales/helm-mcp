package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// wireSchema is the subset of JSON Schema these tests assert on.
//
// mcp.Tool.InputSchema is typed `any` because it is transported as raw JSON,
// so the test decodes it the same way a client would rather than reaching for
// the server-side Go type.
type wireSchema struct {
	Properties map[string]struct {
		Description string `json:"description"`
	} `json:"properties"`
	Required []string `json:"required"`
}

func decodeSchema(t *testing.T, toolName string, raw any) wireSchema {
	t.Helper()

	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("%s: marshal input schema: %v", toolName, err)
	}
	var s wireSchema
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("%s: decode input schema: %v", toolName, err)
	}
	return s
}

// listTools connects an in-memory client to a fresh server and returns every
// registered tool, so tests can assert on the schemas actually published over
// the wire rather than on the Go structs behind them.
func listTools(t *testing.T) []*mcp.Tool {
	t.Helper()

	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ss, err := NewServer("test").Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })

	var tools []*mcp.Tool
	for tool, err := range cs.Tools(ctx, nil) {
		if err != nil {
			t.Fatalf("list tools: %v", err)
		}
		tools = append(tools, tool)
	}
	if len(tools) == 0 {
		t.Fatal("no tools registered")
	}
	return tools
}

// TestToolSchemasHaveDescriptions guards the struct-tag convention.
//
// The input structs are inferred by google/jsonschema-go, which reads the
// `jsonschema` tag as the field description verbatim and ignores
// `jsonschema_description` entirely. Using the wrong tag silently drops every
// argument description from the published schema, and a literal
// `jsonschema:"required"` publishes the description "required" — the field is
// already marked required by the absence of `omitempty` on its json tag.
func TestToolSchemasHaveDescriptions(t *testing.T) {
	for _, tool := range listTools(t) {
		if tool.InputSchema == nil {
			t.Errorf("%s: no input schema", tool.Name)
			continue
		}
		schema := decodeSchema(t, tool.Name, tool.InputSchema)
		if len(schema.Properties) == 0 {
			t.Errorf("%s: input schema has no properties", tool.Name)
			continue
		}
		for name, prop := range schema.Properties {
			switch prop.Description {
			case "":
				t.Errorf("%s.%s: empty description — is the tag `jsonschema_description` instead of `jsonschema`?", tool.Name, name)
			case "required":
				t.Errorf("%s.%s: description is the literal %q — drop the `jsonschema:\"required\"` tag", tool.Name, name, "required")
			}
		}
	}
}

// TestToolSchemasMarkRequiredFields checks the other half of the convention:
// requiredness comes from the json tag, so a field without `omitempty` must
// show up in the schema's required list.
func TestToolSchemasMarkRequiredFields(t *testing.T) {
	// A representative sample across tool packages; each of these is a
	// mandatory argument with no `omitempty` on its json tag.
	want := map[string][]string{
		"helm_install":     {"release_name", "chart"},
		"helm_uninstall":   {"release_name"},
		"helm_rollback":    {"release_name", "revision"},
		"helm_repo_add":    {"name", "url"},
		"helm_search_hub":  {"keyword"},
		"helm_create":      {"name"},
		"helm_show_values": {"chart"},
	}

	byName := make(map[string]*mcp.Tool)
	for _, tool := range listTools(t) {
		byName[tool.Name] = tool
	}

	for toolName, fields := range want {
		tool, ok := byName[toolName]
		if !ok {
			t.Errorf("tool %s not registered", toolName)
			continue
		}
		schema := decodeSchema(t, tool.Name, tool.InputSchema)
		required := make(map[string]bool, len(schema.Required))
		for _, r := range schema.Required {
			required[r] = true
		}
		for _, f := range fields {
			if !required[f] {
				t.Errorf("%s: %q missing from required %v", toolName, f, schema.Required)
			}
		}
	}
}

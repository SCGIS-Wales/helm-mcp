package server

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
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

// TestToolAnnotations checks that every tool carries annotations and that the
// safety-relevant classifications are right.
//
// Without these an agent cannot tell helm_uninstall apart from helm_list, so a
// tool silently losing its annotations is a real regression.
func TestToolAnnotations(t *testing.T) {
	// Tools that can remove or replace live cluster state.
	destructive := map[string]bool{
		"helm_uninstall":        true,
		"helm_rollback":         true,
		"helm_upgrade":          true,
		"helm_repo_remove":      true,
		"helm_plugin_uninstall": true,
	}
	// A sample of tools that must never be reported as read-only.
	mustNotBeReadOnly := []string{
		"helm_install", "helm_upgrade", "helm_uninstall", "helm_rollback",
		"helm_test", "helm_push", "helm_repo_add", "helm_plugin_install",
	}
	// A sample of tools that must be reported as read-only.
	mustBeReadOnly := []string{
		"helm_list", "helm_status", "helm_history", "helm_get_values",
		"helm_get_manifest", "helm_repo_list", "helm_env", "helm_version",
		"helm_template", "helm_lint", "helm_search_hub",
	}

	byName := make(map[string]*mcp.Tool)
	for _, tool := range listTools(t) {
		if tool.Annotations == nil {
			t.Errorf("%s: no annotations", tool.Name)
			continue
		}
		if tool.Annotations.Title == "" {
			t.Errorf("%s: annotations have no title", tool.Name)
		}
		isDestructive := tool.Annotations.DestructiveHint != nil && *tool.Annotations.DestructiveHint
		if want := destructive[tool.Name]; isDestructive != want {
			t.Errorf("%s: destructiveHint = %v, want %v", tool.Name, isDestructive, want)
		}
		if tool.Annotations.ReadOnlyHint && isDestructive {
			t.Errorf("%s: cannot be both read-only and destructive", tool.Name)
		}
		byName[tool.Name] = tool
	}

	for _, name := range mustNotBeReadOnly {
		if tool, ok := byName[name]; ok && tool.Annotations.ReadOnlyHint {
			t.Errorf("%s: readOnlyHint is true but the tool mutates state", name)
		}
	}
	for _, name := range mustBeReadOnly {
		if tool, ok := byName[name]; ok && !tool.Annotations.ReadOnlyHint {
			t.Errorf("%s: readOnlyHint is false but the tool only reads", name)
		}
	}

	// helm_search_hub queries Artifact Hub, so it must declare an open world.
	if tool, ok := byName["helm_search_hub"]; ok {
		if tool.Annotations.OpenWorldHint == nil || !*tool.Annotations.OpenWorldHint {
			t.Error("helm_search_hub: openWorldHint should be true — it queries Artifact Hub")
		}
	}
}

// TestToolOutputSchemas checks that the tools with stable typed results publish
// an outputSchema, and that the free-form ones do not.
//
// A tool gets an outputSchema by declaring a concrete Out type on its handler;
// leaving it as `any` silently drops both the schema and structuredContent, so
// a regression here is invisible without this test.
func TestToolOutputSchemas(t *testing.T) {
	structured := map[string]bool{
		"helm_list": true, "helm_status": true, "helm_history": true,
		"helm_get_metadata": true, "helm_get_values": true,
		"helm_repo_list": true, "helm_search_repo": true, "helm_search_hub": true,
		"helm_plugin_list": true, "helm_env": true, "helm_version": true,
	}
	// These return free-form YAML or text with no useful schema.
	unstructured := []string{
		"helm_template", "helm_lint", "helm_get_manifest", "helm_show_values", "helm_package",
	}

	byName := make(map[string]*mcp.Tool)
	for _, tool := range listTools(t) {
		byName[tool.Name] = tool
		if want := structured[tool.Name]; (tool.OutputSchema != nil) != want {
			t.Errorf("%s: has outputSchema = %v, want %v", tool.Name, tool.OutputSchema != nil, want)
		}
	}

	// The output schemas must not mark anything required at the top level.
	// Error paths return the zero value, which serialises to `{}`; if that
	// failed schema validation the SDK would turn an ordinary tool error into
	// a protocol error.
	for name := range structured {
		tool, ok := byName[name]
		if !ok || tool.OutputSchema == nil {
			continue
		}
		schema := decodeSchema(t, name, tool.OutputSchema)
		if len(schema.Required) > 0 {
			t.Errorf("%s: outputSchema requires %v; every field must be omitempty so the empty error-path result stays valid",
				name, schema.Required)
		}
	}

	for _, name := range unstructured {
		if tool, ok := byName[name]; ok && tool.OutputSchema != nil {
			t.Errorf("%s: unexpectedly has an outputSchema", name)
		}
	}
}

// TestToolListIsDeterministic guards the ordering the 2026-07-28 spec asks for:
// servers SHOULD return tools/list in a deterministic order so clients can
// cache the list and so prompt caches keep hitting.
func TestToolListIsDeterministic(t *testing.T) {
	first := listTools(t)
	second := listTools(t)

	if len(first) != len(second) {
		t.Fatalf("tool count differs between calls: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Name != second[i].Name {
			t.Fatalf("tool order differs at index %d: %q vs %q", i, first[i].Name, second[i].Name)
		}
	}

	if !slices.IsSortedFunc(first, func(a, b *mcp.Tool) int { return strings.Compare(a.Name, b.Name) }) {
		names := make([]string, len(first))
		for i, tool := range first {
			names[i] = tool.Name
		}
		t.Errorf("tools are not in a stable sorted order: %v", names)
	}
}

// TestServerAdvertisesInstructions checks that the server explains itself to
// clients; the instructions are the only server-level guidance an LLM gets.
func TestServerAdvertisesInstructions(t *testing.T) {
	if instructions == "" {
		t.Fatal("instructions must not be empty")
	}
	for _, want := range []string{"helm_version", "destructiveHint", "namespace"} {
		if !strings.Contains(instructions, want) {
			t.Errorf("instructions do not mention %q", want)
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

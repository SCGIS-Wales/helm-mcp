package resilience

import (
	"bytes"
	"encoding/json"
	"strings"
)

// noisyAnnotations lists Kubernetes annotations that add significant size
// but are rarely useful in LLM context.
var noisyAnnotations = []string{
	"kubectl.kubernetes.io/last-applied-configuration",
	"deployment.kubernetes.io/revision",
	"control-plane.alpha.kubernetes.io/leader",
}

// SanitizeManifest strips noisy Kubernetes fields from a YAML manifest string.
// It removes metadata.managedFields and bulky annotations that inflate payloads
// without adding value for LLM interactions.
//
// The function processes the manifest as a stream of YAML documents separated by
// "---" lines, applying field stripping to each document independently.
// If a document cannot be parsed as JSON (after YAML→JSON conversion), it is
// returned unchanged.
func SanitizeManifest(manifest string) string {
	if manifest == "" {
		return manifest
	}

	docs := splitYAMLDocuments(manifest)
	var result strings.Builder
	result.Grow(len(manifest))

	for i, doc := range docs {
		if i > 0 {
			result.WriteString("---\n")
		}
		trimmed := strings.TrimSpace(doc)
		if trimmed == "" || trimmed == "---" {
			result.WriteString(doc)
			continue
		}

		sanitized := sanitizeDocument(trimmed)
		result.WriteString(sanitized)
		if !strings.HasSuffix(sanitized, "\n") {
			result.WriteByte('\n')
		}
	}
	return result.String()
}

// splitYAMLDocuments splits a multi-document YAML string on "---" boundaries.
func splitYAMLDocuments(manifest string) []string {
	var docs []string
	var current strings.Builder

	for _, line := range strings.Split(manifest, "\n") {
		if strings.TrimSpace(line) == "---" {
			docs = append(docs, current.String())
			current.Reset()
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}
	if current.Len() > 0 {
		docs = append(docs, current.String())
	}
	return docs
}

// sanitizeDocument processes a single YAML document by stripping noisy fields.
// It works on the raw YAML text using line-level heuristics to avoid pulling in
// a YAML library as a direct dependency (all YAML libs are transitive-only today).
func sanitizeDocument(doc string) string {
	lines := strings.Split(doc, "\n")
	var out strings.Builder
	out.Grow(len(doc))

	skip := false
	skipIndent := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if !skip {
				out.WriteString(line)
				out.WriteByte('\n')
			}
			continue
		}

		indent := countLeadingSpaces(line)

		// If we're in a skip block and the current line has a deeper indent, skip it.
		if skip {
			if indent > skipIndent {
				continue
			}
			// Same or lesser indent means the block ended.
			skip = false
		}

		// Check if this line starts a block we want to strip.
		if shouldStripBlock(trimmed) {
			skip = true
			skipIndent = indent
			continue
		}

		// Check for noisy annotations within an annotations block.
		if isNoisyAnnotation(trimmed) {
			// Single-line annotation: skip just this line.
			// Multi-line (value on next lines): skip the block.
			if strings.HasSuffix(trimmed, "|") || strings.HasSuffix(trimmed, ">") {
				skip = true
				skipIndent = indent
			}
			continue
		}

		out.WriteString(line)
		out.WriteByte('\n')
	}

	return strings.TrimRight(out.String(), "\n") + "\n"
}

// shouldStripBlock returns true if the YAML key starts a block that should be removed.
func shouldStripBlock(trimmed string) bool {
	return trimmed == "managedFields:" ||
		strings.HasPrefix(trimmed, "managedFields:")
}

// isNoisyAnnotation checks if a line is one of the known noisy annotation keys.
func isNoisyAnnotation(trimmed string) bool {
	for _, ann := range noisyAnnotations {
		// Match "annotation-key: value" or "annotation-key: |"
		if strings.HasPrefix(trimmed, ann+":") {
			return true
		}
	}
	return false
}

// countLeadingSpaces returns the number of leading space characters.
func countLeadingSpaces(s string) int {
	for i, ch := range s {
		if ch != ' ' {
			return i
		}
	}
	return len(s)
}

// SanitizeJSON strips noisy fields from a JSON-encoded Kubernetes object.
// This is useful for tool responses that return structured data rather than
// raw YAML manifests.
func SanitizeJSON(data []byte) []byte {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return data
	}
	stripNoisyFields(obj)
	cleaned, err := json.Marshal(obj)
	if err != nil {
		return data
	}
	// Pretty print for LLM readability.
	var buf bytes.Buffer
	if err := json.Indent(&buf, cleaned, "", "  "); err != nil {
		return cleaned
	}
	return buf.Bytes()
}

// stripNoisyFields recursively removes noisy Kubernetes metadata fields.
func stripNoisyFields(obj map[string]interface{}) {
	// Remove top-level noisy fields.
	if meta, ok := obj["metadata"].(map[string]interface{}); ok {
		delete(meta, "managedFields")

		if ann, ok := meta["annotations"].(map[string]interface{}); ok {
			for _, key := range noisyAnnotations {
				delete(ann, key)
			}
			// Remove empty annotations map.
			if len(ann) == 0 {
				delete(meta, "annotations")
			}
		}
	}

	// Recurse into nested objects (e.g., spec.template.metadata).
	for _, v := range obj {
		switch val := v.(type) {
		case map[string]interface{}:
			stripNoisyFields(val)
		case []interface{}:
			for _, item := range val {
				if m, ok := item.(map[string]interface{}); ok {
					stripNoisyFields(m)
				}
			}
		}
	}
}

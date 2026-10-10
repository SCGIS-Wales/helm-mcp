package helmengine

import (
	"fmt"
	"os"
	"sort"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"

	"helm.sh/helm/v4/pkg/strvals"
)

// MergeInlineValues deep-merges the tool's inline values over the values
// already read from values files, matching how `--set a.b=c` only replaces
// the leaf it names. A shallow merge would silently drop every sibling key
// under a top-level map that the inline values touch.
//
// base is modified in place and returned. A nil base is treated as empty.
func MergeInlineValues(base, inline map[string]interface{}) map[string]interface{} {
	if base == nil {
		base = make(map[string]interface{}, len(inline))
	}
	for k, v := range inline {
		srcMap, srcIsMap := v.(map[string]interface{})
		dstMap, dstIsMap := base[k].(map[string]interface{})
		if srcIsMap && dstIsMap {
			base[k] = MergeInlineValues(dstMap, srcMap)
			continue
		}
		base[k] = v
	}
	return base
}

// ApplySetValues applies --set-string, --set-file and --set-literal entries
// to base, in that order, exactly as Helm's values.Options does after
// --set. Inline values stand in for --set, so callers apply this after
// MergeInlineValues.
//
// --set-file only reads local files: the path is validated by the tool layer
// and fetching URLs here would bypass its SSRF checks.
func ApplySetValues(base map[string]interface{}, opts ValueSetOptions) (map[string]interface{}, error) {
	if base == nil {
		base = map[string]interface{}{}
	}
	for _, v := range opts.SetString {
		if err := strvals.ParseIntoString(v, base); err != nil {
			return nil, fmt.Errorf("failed parsing set_string data: %w", err)
		}
	}
	for _, v := range opts.SetFile {
		reader := func(rs []rune) (any, error) {
			b, err := os.ReadFile(string(rs))
			if err != nil {
				return nil, err
			}
			return string(b), nil
		}
		if err := strvals.ParseIntoFile(v, base, reader); err != nil {
			return nil, fmt.Errorf("failed parsing set_file data: %w", err)
		}
	}
	for _, v := range opts.SetLiteral {
		if err := strvals.ParseLiteralInto(v, base); err != nil {
			return nil, fmt.Errorf("failed parsing set_literal data: %w", err)
		}
	}
	return base, nil
}

// ResourceRefs flattens the resources Helm attaches to a release status
// (keyed by "apiVersion/Kind") into a sorted list of object references.
// Only identifying fields are kept: full objects can run to megabytes.
func ResourceRefs(resources map[string][]runtime.Object) []ResourceRef {
	var refs []ResourceRef
	for _, objs := range resources {
		for _, obj := range objs {
			gvk := obj.GetObjectKind().GroupVersionKind()
			ref := ResourceRef{APIVersion: gvk.GroupVersion().String(), Kind: gvk.Kind}
			if acc, err := meta.Accessor(obj); err == nil {
				ref.Name = acc.GetName()
				ref.Namespace = acc.GetNamespace()
			}
			refs = append(refs, ref)
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		a, b := refs[i], refs[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Namespace != b.Namespace {
			return a.Namespace < b.Namespace
		}
		return a.Name < b.Name
	})
	return refs
}

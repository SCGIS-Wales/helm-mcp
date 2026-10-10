package helmengine

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	releaseutil "helm.sh/helm/v4/pkg/release/v1/util"
)

// HookManifest is a rendered chart hook, independent of the Helm major
// version that produced it.
type HookManifest struct {
	Path     string
	Manifest string
	// IsTest reports whether the hook runs on the "test" event, so
	// skip_tests can drop it the way `helm template --skip-tests` does.
	IsTest bool
}

// manifestSourceRegex extracts the template path (minus the chart name) from
// the "# Source:" comment Helm writes above every rendered document.
var manifestSourceRegex = regexp.MustCompile("# Source: [^/]+/(.+)")

// RenderTemplateOutput assembles `helm template` output from a dry-run
// release the way the Helm CLI does: the manifest, then every hook unless
// hooks are disabled (dropping test hooks when skipTests is set), then the
// show_only filter. Both engines share it so v3 and v4 render identically.
func RenderTemplateOutput(manifest string, hooks []HookManifest, opts *TemplateOptions) (string, error) {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(manifest))
	b.WriteString("\n")

	if !opts.NoHooks {
		for _, h := range hooks {
			if opts.SkipTests && h.IsTest {
				continue
			}
			fmt.Fprintf(&b, "---\n# Source: %s\n%s\n", h.Path, h.Manifest)
		}
	}

	if len(opts.ShowOnly) == 0 {
		return b.String(), nil
	}
	return filterShowOnly(b.String(), opts.ShowOnly)
}

// filterShowOnly keeps only the documents rendered from templates matching
// one of the given paths or globs, and fails if a pattern matches nothing,
// exactly like `helm template --show-only`.
func filterShowOnly(rendered string, showOnly []string) (string, error) {
	split := releaseutil.SplitManifests(rendered)
	keys := make([]string, 0, len(split))
	for k := range split {
		keys = append(keys, k)
	}
	sort.Sort(releaseutil.BySplitManifestsOrder(keys))

	var out strings.Builder
	for _, pattern := range showOnly {
		pattern = filepath.ToSlash(pattern)
		missing := true
		for _, k := range keys {
			doc := split[k]
			m := manifestSourceRegex.FindStringSubmatch(doc)
			if len(m) == 0 {
				continue
			}
			if matched, _ := filepath.Match(pattern, m[1]); !matched {
				continue
			}
			fmt.Fprintf(&out, "---\n%s\n", doc)
			missing = false
		}
		if missing {
			return "", fmt.Errorf("could not find template %s in chart", pattern)
		}
	}
	return out.String(), nil
}

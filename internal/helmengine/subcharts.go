package helmengine

import (
	"os"
	"path/filepath"
	"strings"
)

// ExpandSubchartPaths appends every subchart found under each path's charts/
// directory, both unpacked charts and packaged archives, the way `helm lint
// --with-subcharts` does. The lint action itself ignores WithSubcharts.
func ExpandSubchartPaths(paths []string) []string {
	out := append([]string(nil), paths...)
	for _, p := range paths {
		_ = filepath.Walk(filepath.Join(p, "charts"), func(path string, info os.FileInfo, _ error) error {
			if info == nil {
				return nil
			}
			switch {
			case info.Name() == "Chart.yaml":
				out = append(out, filepath.Dir(path))
			case strings.HasSuffix(path, ".tgz"), strings.HasSuffix(path, ".tar.gz"):
				out = append(out, path)
			}
			return nil
		})
	}
	return out
}

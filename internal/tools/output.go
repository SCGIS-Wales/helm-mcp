package tools

import "github.com/ssddgreg/helm-mcp/internal/helmengine"

// Output types for the tools whose results have a stable shape.
//
// Declaring a concrete Out type on a handler makes the SDK derive an
// outputSchema for the tool and emit structuredContent alongside the existing
// text content, so clients can consume results without parsing prose.
//
// Two conventions matter here:
//
//   - Collections are wrapped in an object rather than returned as a bare
//     array. Objects are what clients and schema tooling handle best, and the
//     wrapper gives somewhere to put the count.
//   - Every field is `omitempty`. google/jsonschema-go marks a field required
//     unless its json tag says omitempty, and the SDK validates the marshalled
//     output against that schema. Error paths return the zero value, which
//     serialises to `{}` — that must stay valid, or a plain tool error would
//     be escalated into a protocol error.
type (
	// ListOutput is the result of helm_list.
	ListOutput struct {
		Releases []*helmengine.ReleaseInfo `json:"releases,omitempty"`
		Count    int                       `json:"count,omitempty"`
	}

	// StatusOutput is the result of helm_status.
	StatusOutput struct {
		Release *helmengine.ReleaseInfo `json:"release,omitempty"`
	}

	// HistoryOutput is the result of helm_history.
	HistoryOutput struct {
		Revisions []*helmengine.ReleaseInfo `json:"revisions,omitempty"`
		Count     int                       `json:"count,omitempty"`
	}

	// MetadataOutput is the result of helm_get_metadata.
	MetadataOutput struct {
		Metadata *helmengine.MetadataInfo `json:"metadata,omitempty"`
	}

	// ValuesOutput is the result of helm_get_values.
	ValuesOutput struct {
		Values map[string]interface{} `json:"values,omitempty"`
	}

	// RepoListOutput is the result of helm_repo_list.
	RepoListOutput struct {
		Repositories []*helmengine.RepoEntry `json:"repositories,omitempty"`
		Count        int                     `json:"count,omitempty"`
	}

	// SearchOutput is the result of helm_search_repo and helm_search_hub.
	SearchOutput struct {
		Results []*helmengine.SearchResult `json:"results,omitempty"`
		Count   int                        `json:"count,omitempty"`
	}

	// PluginListOutput is the result of helm_plugin_list.
	PluginListOutput struct {
		Plugins []*helmengine.PluginInfo `json:"plugins,omitempty"`
		Count   int                      `json:"count,omitempty"`
	}

	// EnvOutput is the result of helm_env.
	EnvOutput struct {
		Environment map[string]string `json:"environment,omitempty"`
	}

	// VersionOutput is the result of helm_version.
	VersionOutput struct {
		Version *helmengine.VersionInfo `json:"version,omitempty"`
	}
)

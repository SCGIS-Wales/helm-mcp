package tools

import (
	"errors"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/security"
)

// ChartSourceInput carries the chart location and authentication flags that
// install, upgrade, template, show and pull share (Helm's ChartPathOptions).
type ChartSourceInput struct {
	Repo                  string `json:"repo,omitempty" jsonschema:"Chart repository URL to look the chart up in, instead of a configured repo (helm --repo)"`
	Username              string `json:"username,omitempty" jsonschema:"Username for the chart repository or OCI registry"`
	Password              string `json:"password,omitempty" jsonschema:"Password for the chart repository or OCI registry"`
	CAFile                string `json:"ca_file,omitempty" jsonschema:"CA bundle to verify the chart repository or registry certificate"`
	CertFile              string `json:"cert_file,omitempty" jsonschema:"Client certificate for the chart repository or registry"`
	KeyFile               string `json:"key_file,omitempty" jsonschema:"Client key for the chart repository or registry"`
	InsecureSkipTLSVerify bool   `json:"insecure_skip_tls_verify,omitempty" jsonschema:"Skip TLS certificate checks when downloading the chart (insecure)"`
	PlainHTTP             bool   `json:"plain_http,omitempty" jsonschema:"Use plain HTTP to reach an OCI registry (insecure)"`
	PassCredentials       bool   `json:"pass_credentials,omitempty" jsonschema:"Send the credentials to every domain the chart download is redirected to"`
	Verify                bool   `json:"verify,omitempty" jsonschema:"Verify the chart provenance (.prov) against the keyring before using it"`
	Keyring               string `json:"keyring,omitempty" jsonschema:"Public keyring used by verify"`
}

// Validate applies the SSRF check to the repo URL and the path checks to
// every certificate and keyring file.
func (c *ChartSourceInput) Validate() error {
	if c.Repo != "" {
		if err := security.ValidateURL(c.Repo); err != nil {
			return err
		}
	}
	return ValidateFilePaths(c.CAFile, c.CertFile, c.KeyFile, c.Keyring)
}

// ToOptions converts the input to engine options. The caller should defer
// ZeroPassword on the result.
func (c *ChartSourceInput) ToOptions() helmengine.ChartSourceOptions {
	return helmengine.ChartSourceOptions{
		Repo:                  c.Repo,
		Username:              c.Username,
		Password:              c.Password,
		CAFile:                c.CAFile,
		CertFile:              c.CertFile,
		KeyFile:               c.KeyFile,
		InsecureSkipTLSVerify: c.InsecureSkipTLSVerify,
		PlainHTTP:             c.PlainHTTP,
		PassCredentialsAll:    c.PassCredentials,
		Verify:                c.Verify,
		Keyring:               c.Keyring,
	}
}

// ValueSetInput carries the typed --set variants. Plain --set is covered by
// the inline values object.
type ValueSetInput struct {
	SetString  []string `json:"set_string,omitempty" jsonschema:"Values as key=value pairs, always treated as strings (helm --set-string)"`
	SetFile    []string `json:"set_file,omitempty" jsonschema:"Values as key=path pairs; each value is the content of that local file (helm --set-file)"`
	SetLiteral []string `json:"set_literal,omitempty" jsonschema:"Values as key=value pairs, taken literally with no escaping or splitting (helm --set-literal)"`
}

// Validate checks every set_file path. Only local files are read, so URLs
// are rejected rather than fetched.
func (v *ValueSetInput) Validate() error {
	for _, p := range helmengine.SetFilePaths(v.SetFile) {
		if isRemoteRef(p) {
			return errors.New("set_file only reads local files; use values_files for URLs")
		}
		if err := ValidateFilePaths(p); err != nil {
			return err
		}
	}
	return nil
}

// ToOptions converts the input to engine options.
func (v *ValueSetInput) ToOptions() helmengine.ValueSetOptions {
	return helmengine.ValueSetOptions{
		SetString:  v.SetString,
		SetFile:    v.SetFile,
		SetLiteral: v.SetLiteral,
	}
}

// PostRenderInput selects a post-renderer. In Helm v4 a post-renderer is an
// installed plugin, referenced by name. Helm v3 post-renderers run an
// arbitrary executable, so the v3 engine refuses them.
type PostRenderInput struct {
	PostRenderer     string   `json:"post_renderer,omitempty" jsonschema:"Name of an installed post-renderer plugin applied to the rendered manifests (v4 only)"`
	PostRendererArgs []string `json:"post_renderer_args,omitempty" jsonschema:"Arguments passed to the post-renderer plugin (v4 only)"`
}

// Validate checks the plugin name.
func (p *PostRenderInput) Validate() error {
	if p.PostRenderer == "" {
		return nil
	}
	return security.ValidatePluginName(p.PostRenderer)
}

// ToOptions converts the input to engine options.
func (p *PostRenderInput) ToOptions() helmengine.PostRenderOptions {
	return helmengine.PostRenderOptions{
		PostRenderer:     p.PostRenderer,
		PostRendererArgs: p.PostRendererArgs,
	}
}

// ValidateReleaseInputs validates the shared inputs of tools that render a
// chart (install, upgrade, template).
func ValidateReleaseInputs(src *ChartSourceInput, sets *ValueSetInput, pr *PostRenderInput) error {
	if err := src.Validate(); err != nil {
		return err
	}
	if err := sets.Validate(); err != nil {
		return err
	}
	return pr.Validate()
}

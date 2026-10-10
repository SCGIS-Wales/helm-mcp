package repo

import (
	"context"

	"github.com/ssddgreg/helm-mcp/internal/helmengine"
	"github.com/ssddgreg/helm-mcp/internal/security"
	"github.com/ssddgreg/helm-mcp/internal/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type AddInput struct {
	tools.GlobalInput
	Name            string `json:"name" jsonschema:"Repository name"`
	URL             string `json:"url" jsonschema:"Repository URL"`
	Username        string `json:"username,omitempty" jsonschema:"Repository username"`
	Password        string `json:"password,omitempty" jsonschema:"Repository password"`
	ForceUpdate     bool   `json:"force_update,omitempty" jsonschema:"Replace existing entry"`
	CAFile          string `json:"ca_file,omitempty" jsonschema:"CA bundle file"`
	CertFile        string `json:"cert_file,omitempty" jsonschema:"Client certificate for mutual TLS"`
	KeyFile         string `json:"key_file,omitempty" jsonschema:"Client key for mutual TLS"`
	InsecureSkipTLS bool   `json:"insecure_skip_tls,omitempty" jsonschema:"Skip TLS verification"`
	PassCredentials bool   `json:"pass_credentials,omitempty" jsonschema:"Send the credentials to every domain the repository redirects to"`
}

var AddTool = &mcp.Tool{
	Name:        "helm_repo_add",
	Description: "Add a chart repository.",
	Annotations: tools.Mutating("Add a repository", true),
}

func HandleAdd(ctx context.Context, _ *mcp.CallToolRequest, input AddInput) (*mcp.CallToolResult, any, error) {
	if err := tools.ValidateGlobalInput(&input.GlobalInput); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := security.ValidateRepoName(input.Name); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := tools.ValidateFilePaths(input.CAFile, input.CertFile, input.KeyFile); err != nil {
		return tools.ErrorResult(err), nil, nil
	}
	if err := security.ValidateURL(input.URL); err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	engine := tools.SelectEngine(input.HelmVersion)
	defer input.ZeroBearerToken()

	opts := &helmengine.RepoAddOptions{
		Name:                  input.Name,
		URL:                   input.URL,
		Username:              input.Username,
		Password:              input.Password,
		ForceUpdate:           input.ForceUpdate,
		CAFile:                input.CAFile,
		CertFile:              input.CertFile,
		KeyFile:               input.KeyFile,
		InsecureSkipTLSVerify: input.InsecureSkipTLS,
		PassCredentialsAll:    input.PassCredentials,
	}
	defer opts.ZeroPassword()

	err := engine.RepoAdd(ctx, opts)
	if err != nil {
		return tools.ErrorResult(err), nil, nil
	}

	return tools.TextResult("Repository added successfully"), nil, nil
}

package v4

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/postrenderer"
	"helm.sh/helm/v4/pkg/registry"
)

// registryClientSetter is implemented by every action that locates charts
// (Install, Upgrade, Show, Pull).
type registryClientSetter interface {
	SetRegistryClient(*registry.Client)
}

// applyChartSource copies the chart source arguments onto an action's
// ChartPathOptions. When they change how an OCI registry is reached
// (credentials, TLS or plain HTTP) the action also gets a registry client
// built with them, as the Helm CLI builds one per command.
//
// It returns the registry client the action will use, so that dependency
// downloads go through the same client.
func applyChartSource(cpo *action.ChartPathOptions, setter registryClientSetter, src *helmengine.ChartSourceOptions, settings *cli.EnvSettings, current *registry.Client) (*registry.Client, error) {
	cpo.RepoURL = src.Repo
	cpo.Username = src.Username
	cpo.Password = src.Password
	cpo.CaFile = src.CAFile
	cpo.CertFile = src.CertFile
	cpo.KeyFile = src.KeyFile
	cpo.InsecureSkipTLSVerify = src.InsecureSkipTLSVerify
	cpo.PlainHTTP = src.PlainHTTP
	cpo.PassCredentialsAll = src.PassCredentialsAll
	cpo.Verify = src.Verify
	cpo.Keyring = src.Keyring

	if !needsCustomRegistryClient(src) {
		return current, nil
	}
	rc, err := newRegistryClient(settings, src)
	if err != nil {
		return nil, err
	}
	setter.SetRegistryClient(rc)
	return rc, nil
}

func needsCustomRegistryClient(src *helmengine.ChartSourceOptions) bool {
	return src.Username != "" || src.Password != "" || src.CAFile != "" ||
		src.CertFile != "" || src.KeyFile != "" || src.InsecureSkipTLSVerify || src.PlainHTTP
}

// newRegistryClient mirrors the Helm CLI's newRegistryClient: a custom TLS
// transport when certificates or insecure mode are requested, otherwise the
// default transport (optionally over plain HTTP).
func newRegistryClient(settings *cli.EnvSettings, src *helmengine.ChartSourceOptions) (*registry.Client, error) {
	opts := []registry.ClientOption{
		registry.ClientOptDebug(settings.Debug),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(io.Discard),
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
		registry.ClientOptBasicAuth(src.Username, src.Password),
	}

	switch {
	case (src.CertFile != "" && src.KeyFile != "") || src.CAFile != "" || src.InsecureSkipTLSVerify:
		tlsConf, err := helmengine.NewClientTLSConfig(src.CertFile, src.KeyFile, src.CAFile, src.InsecureSkipTLSVerify)
		if err != nil {
			return nil, fmt.Errorf("can't create TLS config for registry client: %w", err)
		}
		opts = append(opts, registry.ClientOptHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConf,
				Proxy:           http.ProxyFromEnvironment,
			},
		}))
	case src.PlainHTTP:
		opts = append(opts, registry.ClientOptPlainHTTP())
	}

	rc, err := registry.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry client: %w", err)
	}
	return rc, nil
}

// newPostRenderer resolves a post-renderer. In Helm v4 a post-renderer is an
// installed plugin, referenced by name; it never executes an arbitrary path.
func newPostRenderer(settings *cli.EnvSettings, opts *helmengine.PostRenderOptions) (postrenderer.PostRenderer, error) {
	if opts.PostRenderer == "" {
		if len(opts.PostRendererArgs) > 0 {
			return nil, errors.New("post_renderer_args requires post_renderer")
		}
		return nil, nil
	}
	pr, err := postrenderer.NewPostRendererPlugin(settings, opts.PostRenderer, opts.PostRendererArgs...)
	if err != nil {
		return nil, fmt.Errorf("invalid post_renderer %q: %w", opts.PostRenderer, err)
	}
	return pr, nil
}

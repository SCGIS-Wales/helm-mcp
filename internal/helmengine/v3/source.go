package v3

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
)

// defaultKeyring mirrors the Helm v3 CLI default for --keyring.
func defaultKeyring() string {
	return filepath.Join(helmengine.GnuPGHome(), "pubring.gpg")
}

// registryClientSetter is implemented by every action that locates charts
// (Install, Upgrade, Show, Pull).
type registryClientSetter interface {
	SetRegistryClient(*registry.Client)
}

// applyChartSource copies the chart source arguments onto an action's
// ChartPathOptions and, when they change how an OCI registry is reached,
// gives the action a registry client built with them. It returns the
// registry client the action will use.
func applyChartSource(cpo *action.ChartPathOptions, setter registryClientSetter, src *helmengine.ChartSourceOptions, settings *cli.EnvSettings, current *registry.Client) (*registry.Client, error) {
	cpo.RepoURL = src.Repo
	cpo.Username = src.Username
	cpo.Password = src.Password
	cpo.CaFile = src.CAFile
	cpo.CertFile = src.CertFile
	cpo.KeyFile = src.KeyFile
	cpo.InsecureSkipTLSverify = src.InsecureSkipTLSVerify
	cpo.PlainHTTP = src.PlainHTTP
	cpo.PassCredentialsAll = src.PassCredentialsAll
	cpo.Verify = src.Verify
	cpo.Keyring = src.Keyring
	if cpo.Verify && cpo.Keyring == "" {
		cpo.Keyring = defaultKeyring()
	}

	if src.Username == "" && src.Password == "" && src.CAFile == "" && src.CertFile == "" &&
		src.KeyFile == "" && !src.InsecureSkipTLSVerify && !src.PlainHTTP {
		return current, nil
	}

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
	setter.SetRegistryClient(rc)
	return rc, nil
}

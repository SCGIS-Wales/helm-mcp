package v4

import (
	"context"
	"fmt"
	"io"
	"os"

	helmengine "github.com/ssddgreg/helm-mcp/internal/helmengine"

	"helm.sh/helm/v4/pkg/registry"
)

func (e *V4Engine) RegistryLogin(_ context.Context, opts *helmengine.RegistryLoginOptions) error {
	client, err := registry.NewClient(
		registry.ClientOptWriter(io.Discard),
		registry.ClientOptCredentialsFile(os.Getenv("HELM_REGISTRY_CONFIG")),
	)
	if err != nil {
		return fmt.Errorf("failed to create registry client: %w", err)
	}

	loginOpts := []registry.LoginOption{
		registry.LoginOptBasicAuth(opts.Username, opts.Password),
		registry.LoginOptInsecure(opts.Insecure),
		registry.LoginOptPlainText(opts.PlainHTTP),
	}
	// ca_file, cert_file and key_file used to be accepted and dropped. The
	// configuration is built here, not by LoginOptTLSClientConfig, because
	// that option panics on an unreadable file and would take the whole
	// server down.
	if opts.CAFile != "" || opts.CertFile != "" || opts.KeyFile != "" {
		tlsConf, err := helmengine.NewClientTLSConfig(opts.CertFile, opts.KeyFile, opts.CAFile, opts.Insecure)
		if err != nil {
			return fmt.Errorf("helm registry login: %w", err)
		}
		loginOpts = append(loginOpts, registry.LoginOptTLSClientConfigFromConfig(tlsConf))
	}
	err = client.Login(opts.Hostname, loginOpts...)
	if err != nil {
		return fmt.Errorf("helm registry login failed: %w", err)
	}

	return nil
}

func (e *V4Engine) RegistryLogout(_ context.Context, opts *helmengine.RegistryLogoutOptions) error {
	client, err := registry.NewClient(
		registry.ClientOptWriter(io.Discard),
		registry.ClientOptCredentialsFile(os.Getenv("HELM_REGISTRY_CONFIG")),
	)
	if err != nil {
		return fmt.Errorf("failed to create registry client: %w", err)
	}

	err = client.Logout(opts.Hostname)
	if err != nil {
		return fmt.Errorf("helm registry logout failed: %w", err)
	}

	return nil
}

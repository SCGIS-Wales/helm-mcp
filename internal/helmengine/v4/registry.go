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

	err = client.Login(
		opts.Hostname,
		registry.LoginOptBasicAuth(opts.Username, opts.Password),
		registry.LoginOptInsecure(opts.Insecure),
	)
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

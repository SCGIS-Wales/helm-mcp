package helmengine

import (
	"context"
	"testing"
)

func TestMockEngineImplementsInterface(t *testing.T) {
	var _ Engine = &MockEngine{}
}

func TestDefaultRelease(t *testing.T) {
	r := DefaultRelease()
	if r.Name != "my-release" {
		t.Errorf("expected name 'my-release', got %q", r.Name)
	}
	if r.Status != "deployed" {
		t.Errorf("expected status 'deployed', got %q", r.Status)
	}
	if r.Revision != 1 {
		t.Errorf("expected revision 1, got %d", r.Revision)
	}
}

func TestMockEngineCallTracking(t *testing.T) {
	m := &MockEngine{}
	ctx := context.Background()

	cfg := &GlobalConfig{Namespace: "test-ns"}
	_, _ = m.Install(ctx, cfg, &InstallOptions{ReleaseName: "test", Chart: "nginx"})
	if m.LastInstallOpts == nil || m.LastInstallOpts.ReleaseName != "test" {
		t.Error("expected install opts to be tracked")
	}
	if m.LastConfig == nil || m.LastConfig.Namespace != "test-ns" {
		t.Error("expected config to be tracked")
	}
}

func TestMockEngineCustomFunctions(t *testing.T) {
	m := &MockEngine{
		ListFn: func(ctx context.Context, cfg *GlobalConfig, opts *ListOptions) ([]*ReleaseInfo, error) {
			return []*ReleaseInfo{
				{Name: "custom-release", Status: "failed"},
			}, nil
		},
	}

	result, err := m.List(context.Background(), &GlobalConfig{}, &ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Name != "custom-release" {
		t.Errorf("expected custom result, got %v", result)
	}
}

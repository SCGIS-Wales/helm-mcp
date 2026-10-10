package v3

import (
	"io"
	"testing"

	"helm.sh/helm/v3/pkg/action"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

func TestUpgradeShouldInstall(t *testing.T) {
	cfg := &action.Configuration{
		Releases:   storage.Init(driver.NewMemory()),
		KubeClient: &kubefake.PrintingKubeClient{Out: io.Discard},
		Log:        func(string, ...interface{}) {},
	}
	check := func(wantInstall, wantReplace bool) {
		t.Helper()
		install, replace, err := upgradeShouldInstall(cfg, "app")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if install != wantInstall || replace != wantReplace {
			t.Errorf("got install=%v replace=%v, want install=%v replace=%v", install, replace, wantInstall, wantReplace)
		}
	}
	store := func(version int, status release.Status) {
		t.Helper()
		rel := &release.Release{Name: "app", Namespace: "default", Version: version, Info: &release.Info{Status: status}}
		if err := cfg.Releases.Create(rel); err != nil {
			t.Fatal(err)
		}
	}

	check(true, false)
	store(1, release.StatusDeployed)
	check(false, false)
	store(2, release.StatusUninstalled)
	check(true, true)
}

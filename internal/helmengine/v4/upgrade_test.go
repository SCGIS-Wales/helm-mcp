package v4

import (
	"io"
	"testing"

	"helm.sh/helm/v4/pkg/action"
	kubefake "helm.sh/helm/v4/pkg/kube/fake"
	"helm.sh/helm/v4/pkg/release/common"
	v1release "helm.sh/helm/v4/pkg/release/v1"
	"helm.sh/helm/v4/pkg/storage"
	"helm.sh/helm/v4/pkg/storage/driver"
)

// TestUpgradeShouldInstall checks the `upgrade --install` decision against
// in-memory release storage, mirroring the Helm CLI: install when there is
// no history, install with replace when the latest revision is uninstalled,
// upgrade otherwise.
func TestUpgradeShouldInstall(t *testing.T) {
	cfg := &action.Configuration{
		Releases:   storage.Init(driver.NewMemory()),
		KubeClient: &kubefake.PrintingKubeClient{Out: io.Discard},
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
	store := func(version int, status common.Status) {
		t.Helper()
		rel := &v1release.Release{Name: "app", Namespace: "default", Version: version, Info: &v1release.Info{Status: status}}
		if err := cfg.Releases.Create(rel); err != nil {
			t.Fatal(err)
		}
	}

	check(true, false)
	store(1, common.StatusSuperseded)
	store(2, common.StatusDeployed)
	check(false, false)
	store(3, common.StatusUninstalled)
	check(true, true)
}

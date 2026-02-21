package helmengine

import (
	"encoding/json"
	"testing"
	"time"
)

func TestReleaseInfoJSON(t *testing.T) {
	info := &ReleaseInfo{
		Name:         "my-release",
		Namespace:    "default",
		Revision:     3,
		Status:       "deployed",
		Chart:        "nginx",
		ChartVersion: "1.2.3",
		AppVersion:   "1.25.0",
		Updated:      time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
		Labels:       map[string]string{"env": "prod"},
	}

	b, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal ReleaseInfo: %v", err)
	}

	var decoded ReleaseInfo
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ReleaseInfo: %v", err)
	}

	if decoded.Name != "my-release" {
		t.Errorf("Name = %q, want %q", decoded.Name, "my-release")
	}
	if decoded.Revision != 3 {
		t.Errorf("Revision = %d, want 3", decoded.Revision)
	}
	if decoded.Status != "deployed" {
		t.Errorf("Status = %q, want %q", decoded.Status, "deployed")
	}
	if decoded.Labels["env"] != "prod" {
		t.Errorf("Labels[env] = %q, want %q", decoded.Labels["env"], "prod")
	}
}

func TestReleaseDetailJSON(t *testing.T) {
	detail := &ReleaseDetail{
		Release: &ReleaseInfo{
			Name:      "my-release",
			Namespace: "default",
		},
		Values:   map[string]interface{}{"replicas": 3},
		Manifest: "---\napiVersion: v1\nkind: Service",
		Hooks:    "---\nhook content",
		Notes:    "Release notes here",
	}

	b, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded ReleaseDetail
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Release.Name != "my-release" {
		t.Errorf("Release.Name = %q, want %q", decoded.Release.Name, "my-release")
	}
	if decoded.Notes != "Release notes here" {
		t.Errorf("Notes = %q, want %q", decoded.Notes, "Release notes here")
	}
}

func TestMetadataInfoJSON(t *testing.T) {
	md := &MetadataInfo{
		Name:         "my-release",
		Namespace:    "default",
		Revision:     2,
		Status:       "deployed",
		Chart:        "nginx",
		ChartVersion: "1.0.0",
		AppVersion:   "1.25.0",
		DeployedAt:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	}

	b, err := json.Marshal(md)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded MetadataInfo
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ChartVersion != "1.0.0" {
		t.Errorf("ChartVersion = %q, want %q", decoded.ChartVersion, "1.0.0")
	}
}

func TestLintResult(t *testing.T) {
	result := &LintResult{
		TotalCharts: 2,
		Passed:      false,
		Messages: []LintMessage{
			{Severity: "ERROR", Path: "templates/deployment.yaml", Message: "missing required field"},
			{Severity: "WARNING", Path: "Chart.yaml", Message: "icon not found"},
			{Severity: "INFO", Path: "", Message: "chart lint complete"},
		},
	}

	if result.TotalCharts != 2 {
		t.Errorf("TotalCharts = %d, want 2", result.TotalCharts)
	}
	if result.Passed {
		t.Error("expected Passed=false")
	}
	if len(result.Messages) != 3 {
		t.Fatalf("Messages len = %d, want 3", len(result.Messages))
	}
	if result.Messages[0].Severity != "ERROR" {
		t.Errorf("Messages[0].Severity = %q, want %q", result.Messages[0].Severity, "ERROR")
	}
}

func TestInstallOptionsJSON(t *testing.T) {
	opts := &InstallOptions{
		ReleaseName:     "my-release",
		Chart:           "bitnami/nginx",
		Version:         ">=1.0.0",
		CreateNamespace: true,
		Wait:            true,
		WaitForJobs:     true,
		Timeout:         "5m0s",
		DryRun:          "client",
		Labels:          map[string]string{"team": "platform"},
	}

	b, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded InstallOptions
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.ReleaseName != "my-release" {
		t.Errorf("ReleaseName = %q, want %q", decoded.ReleaseName, "my-release")
	}
	if !decoded.CreateNamespace {
		t.Error("expected CreateNamespace=true")
	}
	if decoded.DryRun != "client" {
		t.Errorf("DryRun = %q, want %q", decoded.DryRun, "client")
	}
	if decoded.Labels["team"] != "platform" {
		t.Errorf("Labels[team] = %q, want %q", decoded.Labels["team"], "platform")
	}
}

func TestUpgradeOptionsJSON(t *testing.T) {
	opts := &UpgradeOptions{
		ReleaseName:     "my-release",
		Chart:           "bitnami/nginx",
		Install:         true,
		Force:           true,
		ResetValues:     false,
		ReuseValues:     true,
		Wait:            true,
		Timeout:         "10m",
		ServerSideApply: true,
	}

	b, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded UpgradeOptions
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !decoded.Install {
		t.Error("expected Install=true")
	}
	if !decoded.Force {
		t.Error("expected Force=true")
	}
	if !decoded.ServerSideApply {
		t.Error("expected ServerSideApply=true")
	}
}

func TestUninstallOptionsJSON(t *testing.T) {
	opts := &UninstallOptions{
		ReleaseName:  "old-release",
		KeepHistory:  true,
		DryRun:       true,
		DisableHooks: false,
		Cascade:      "background",
	}

	b, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded UninstallOptions
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !decoded.KeepHistory {
		t.Error("expected KeepHistory=true")
	}
	if decoded.Cascade != "background" {
		t.Errorf("Cascade = %q, want %q", decoded.Cascade, "background")
	}
}

func TestRepoAddOptionsJSON(t *testing.T) {
	opts := &RepoAddOptions{
		Name:                  "bitnami",
		URL:                   "https://charts.bitnami.com/bitnami",
		Username:              "user",
		Password:              "pass",
		InsecureSkipTLSVerify: true,
		PassCredentialsAll:    false,
	}

	b, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded RepoAddOptions
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Name != "bitnami" {
		t.Errorf("Name = %q, want %q", decoded.Name, "bitnami")
	}
	if !decoded.InsecureSkipTLSVerify {
		t.Error("expected InsecureSkipTLSVerify=true")
	}
}

func TestSearchRepoOptionsJSON(t *testing.T) {
	opts := &SearchRepoOptions{
		Keyword:           "nginx",
		Versions:          true,
		Devel:             false,
		VersionConstraint: ">=1.0.0",
	}

	b, err := json.Marshal(opts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded SearchRepoOptions
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Keyword != "nginx" {
		t.Errorf("Keyword = %q, want %q", decoded.Keyword, "nginx")
	}
	if !decoded.Versions {
		t.Error("expected Versions=true")
	}
}

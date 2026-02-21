package helmengine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchArtifactHub_EmptyKeyword(t *testing.T) {
	results, err := SearchArtifactHub(context.Background(), &SearchHubOptions{Keyword: ""})
	if err == nil {
		t.Fatal("expected error for empty keyword")
	}
	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
}

func TestSearchArtifactHub_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := SearchArtifactHub(ctx, &SearchHubOptions{Keyword: "nginx"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestSearchArtifactHub_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Packages []artifactHubPackage `json:"packages"`
		}{
			Packages: []artifactHubPackage{
				{
					Name:        "nginx",
					Version:     "15.0.0",
					AppVersion:  "1.25.0",
					Description: "NGINX web server",
					Repository: struct {
						Name string `json:"name"`
						URL  string `json:"url"`
					}{"bitnami", "https://charts.bitnami.com/bitnami"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer srv.Close()

	old := artifactHubAPIBase
	setArtifactHubAPIBase(srv.URL)
	defer setArtifactHubAPIBase(old)

	results, err := SearchArtifactHub(context.Background(), &SearchHubOptions{Keyword: "nginx"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "bitnami/nginx" {
		t.Errorf("Name = %q, want %q", results[0].Name, "bitnami/nginx")
	}
	if results[0].ChartVersion != "15.0.0" {
		t.Errorf("ChartVersion = %q, want %q", results[0].ChartVersion, "15.0.0")
	}
	if results[0].URL != "" {
		t.Errorf("URL = %q, want empty (ListRepoURL=false)", results[0].URL)
	}
}

func TestSearchArtifactHub_ListRepoURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Packages []artifactHubPackage `json:"packages"`
		}{
			Packages: []artifactHubPackage{
				{
					Name:    "nginx",
					Version: "15.0.0",
					Repository: struct {
						Name string `json:"name"`
						URL  string `json:"url"`
					}{"bitnami", "https://charts.bitnami.com/bitnami"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer srv.Close()

	old := artifactHubAPIBase
	setArtifactHubAPIBase(srv.URL)
	defer setArtifactHubAPIBase(old)

	results, err := SearchArtifactHub(context.Background(), &SearchHubOptions{
		Keyword:     "nginx",
		ListRepoURL: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].URL != "https://charts.bitnami.com/bitnami" {
		t.Errorf("URL = %q, want %q", results[0].URL, "https://charts.bitnami.com/bitnami")
	}
}

func TestSearchArtifactHub_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error")) //nolint:errcheck
	}))
	defer srv.Close()

	old := artifactHubAPIBase
	setArtifactHubAPIBase(srv.URL)
	defer setArtifactHubAPIBase(old)

	_, err := SearchArtifactHub(context.Background(), &SearchHubOptions{Keyword: "nginx"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestSearchArtifactHub_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json")) //nolint:errcheck
	}))
	defer srv.Close()

	old := artifactHubAPIBase
	setArtifactHubAPIBase(srv.URL)
	defer setArtifactHubAPIBase(old)

	_, err := SearchArtifactHub(context.Background(), &SearchHubOptions{Keyword: "nginx"})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSearchArtifactHub_EmptyResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Packages []artifactHubPackage `json:"packages"`
		}{Packages: []artifactHubPackage{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer srv.Close()

	old := artifactHubAPIBase
	setArtifactHubAPIBase(srv.URL)
	defer setArtifactHubAPIBase(old)

	results, err := SearchArtifactHub(context.Background(), &SearchHubOptions{Keyword: "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchArtifactHub_NoRepoName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Packages []artifactHubPackage `json:"packages"`
		}{
			Packages: []artifactHubPackage{
				{
					Name:    "standalone",
					Version: "1.0.0",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer srv.Close()

	old := artifactHubAPIBase
	setArtifactHubAPIBase(srv.URL)
	defer setArtifactHubAPIBase(old)

	results, err := SearchArtifactHub(context.Background(), &SearchHubOptions{Keyword: "standalone"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "standalone" {
		t.Errorf("Name = %q, want %q (no repo prefix)", results[0].Name, "standalone")
	}
}

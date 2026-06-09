package schemagen

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckDriftRequiresTokenWhenConfigured(t *testing.T) {
	err := CheckDrift(fixtureConfig(), DriftOptions{RequireToken: true})
	if err == nil {
		t.Fatal("CheckDrift succeeded without required token")
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN is required") {
		t.Fatalf("CheckDrift error = %q, want GITHUB_TOKEN requirement", err)
	}
}

func TestCheckDriftDetectsPinnedCommitMismatch(t *testing.T) {
	cfg := fixtureConfig()
	localCRD := readFixtureCRD(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/crossplane/crossplane/git/ref/tags/v1.20.7", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"object":{"sha":"0000000000000000000000000000000000000000"}}`)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/tags", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"name":"v1.20.7"}]`)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/contents/cluster/crds/apiextensions.crossplane.io_compositions.yaml", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(localCRD)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	err := CheckDrift(cfg, DriftOptions{GitHubBaseURL: server.URL, HTTPClient: server.Client()})
	if err == nil {
		t.Fatal("CheckDrift succeeded with mismatched pinned commit")
	}
	if !strings.Contains(err.Error(), "pinned release v1.20.7 resolves to different") {
		t.Fatalf("CheckDrift error = %q, want pinned commit mismatch", err)
	}
}

func TestCheckDriftResolvesAnnotatedTagToCommit(t *testing.T) {
	cfg := fixtureConfig()
	localCRD := readFixtureCRD(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/crossplane/crossplane/git/ref/tags/v1.20.7", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"object":{"type":"tag","sha":"annotated-tag-sha"}}`)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/git/tags/annotated-tag-sha", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"object":{"type":"commit","sha":%q}}`, cfg.Releases[0].Commit)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/tags", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"name":"v1.20.7"}]`)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/contents/cluster/crds/apiextensions.crossplane.io_compositions.yaml", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(localCRD)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	if err := CheckDrift(cfg, DriftOptions{GitHubBaseURL: server.URL, HTTPClient: server.Client()}); err != nil {
		t.Fatalf("CheckDrift: %v", err)
	}
}

func TestCheckDriftReportsNetworkErrorsSeparately(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "try later", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := CheckDrift(fixtureConfig(), DriftOptions{GitHubBaseURL: server.URL, HTTPClient: server.Client()})
	if err == nil {
		t.Fatal("CheckDrift succeeded with upstream 503")
	}
	if !strings.Contains(err.Error(), "query upstream drift") {
		t.Fatalf("CheckDrift error = %q, want upstream query context", err)
	}
}

func TestCheckDriftDetectsNewerStableRelease(t *testing.T) {
	cfg := fixtureConfig()
	localCRD := readFixtureCRD(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/crossplane/crossplane/git/ref/tags/v1.20.7", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"object":{"sha":%q}}`, cfg.Releases[0].Commit)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/tags", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"name":"v9.0.0-rc.1"},{"name":"v1.20.8"},{"name":"v1.20.7"}]`)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/contents/cluster/crds/apiextensions.crossplane.io_compositions.yaml", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(localCRD)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	err := CheckDrift(cfg, DriftOptions{GitHubBaseURL: server.URL, HTTPClient: server.Client()})
	if err == nil {
		t.Fatal("CheckDrift succeeded with newer stable release")
	}
	if !strings.Contains(err.Error(), "newer stable Crossplane release v1.20.8") {
		t.Fatalf("CheckDrift error = %q, want newer stable release drift", err)
	}
	if strings.Contains(err.Error(), "v9.0.0-rc.1") {
		t.Fatalf("CheckDrift error = %q, prerelease should not be reported as stable", err)
	}
}

func TestCheckDriftAggregatesFreshnessAndCRDContentDrift(t *testing.T) {
	cfg := fixtureConfig()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/crossplane/crossplane/git/ref/tags/v1.20.7", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"object":{"sha":%q}}`, cfg.Releases[0].Commit)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/tags", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"name":"v1.20.8"},{"name":"v1.20.7"}]`)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/contents/cluster/crds/apiextensions.crossplane.io_compositions.yaml", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "different upstream CRD")
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	err := CheckDrift(cfg, DriftOptions{GitHubBaseURL: server.URL, HTTPClient: server.Client()})
	if err == nil {
		t.Fatal("CheckDrift succeeded with newer stable release and CRD content drift")
	}
	got := err.Error()
	for _, want := range []string{
		"schema upstream drift detected",
		"newer stable Crossplane release v1.20.8",
		"CRD content drift",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("CheckDrift error = %q, want %q", got, want)
		}
	}
}

func TestCheckDriftDetectsCRDContentMismatch(t *testing.T) {
	cfg := fixtureConfig()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/crossplane/crossplane/git/ref/tags/v1.20.7", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"object":{"sha":%q}}`, cfg.Releases[0].Commit)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/tags", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"name":"v1.20.7"}]`)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/contents/cluster/crds/apiextensions.crossplane.io_compositions.yaml", func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("ref"), "v1.20.7"; got != want {
			t.Fatalf("content ref = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Accept"), "application/vnd.github.raw"; got != want {
			t.Fatalf("Accept = %q, want %q", got, want)
		}
		fmt.Fprint(w, "different upstream CRD")
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	err := CheckDrift(cfg, DriftOptions{GitHubBaseURL: server.URL, HTTPClient: server.Client()})
	if err == nil {
		t.Fatal("CheckDrift succeeded with CRD content mismatch")
	}
	if !strings.Contains(err.Error(), "CRD content drift") {
		t.Fatalf("CheckDrift error = %q, want CRD content drift", err)
	}
	if !strings.Contains(err.Error(), "cluster/crds/apiextensions.crossplane.io_compositions.yaml") {
		t.Fatalf("CheckDrift error = %q, want upstream CRD path", err)
	}
}

func TestCheckDriftPassesWhenFixtureMatchesUpstream(t *testing.T) {
	cfg := fixtureConfig()
	localCRD := readFixtureCRD(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/crossplane/crossplane/git/ref/tags/v1.20.7", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"object":{"sha":%q}}`, cfg.Releases[0].Commit)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/tags", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"name":"v1.21.0-rc.1"},{"name":"v1.20.7"}]`)
	})
	mux.HandleFunc("/repos/crossplane/crossplane/contents/cluster/crds/apiextensions.crossplane.io_compositions.yaml", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(localCRD)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	if err := CheckDrift(cfg, DriftOptions{GitHubBaseURL: server.URL, HTTPClient: server.Client()}); err != nil {
		t.Fatalf("CheckDrift: %v", err)
	}
}

func TestUpstreamCRDPathPreservesRealConfigPrefix(t *testing.T) {
	release := ReleaseConfig{
		Tag:       "v2.2.1",
		RawCRDDir: filepath.Join("internal", "analyzer", "schemadata", "upstream", "crossplane", "v2.2.1", "cluster", "crds"),
	}
	localPath := filepath.Join(release.RawCRDDir, "apiextensions.crossplane.io_compositions.yaml")

	got := upstreamCRDPath(release, localPath, nil)
	want := "cluster/crds/apiextensions.crossplane.io_compositions.yaml"
	if got != want {
		t.Fatalf("upstreamCRDPath() = %q, want %q", got, want)
	}
}

func readFixtureCRD(t *testing.T) []byte {
	t.Helper()
	localCRD, err := os.ReadFile(filepath.Join("testdata", "composition-crd.yaml"))
	if err != nil {
		t.Fatalf("read fixture CRD: %v", err)
	}
	return localCRD
}

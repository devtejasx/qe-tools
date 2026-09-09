package utils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPClientHasTimeout(t *testing.T) {
	client := NewHTTPClient()

	if client.Timeout != DefaultHTTPTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultHTTPTimeout, client.Timeout)
	}
	if http.DefaultClient.Timeout != 0 {
		t.Fatal("test assumption broken: http.DefaultClient is expected to have no timeout")
	}
}

func TestNewHTTPClientTimesOut(t *testing.T) {
	blocked := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
	}))
	defer func() {
		close(blocked)
		server.Close()
	}()

	client := NewHTTPClient()
	client.Timeout = 100 * time.Millisecond

	start := time.Now()
	//nolint:bodyclose // the request never completes, there is no body to close
	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("expected the request to a non-responding server to fail")
	}

	var netErr interface{ Timeout() bool }
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Errorf("expected a timeout error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("request took %v, the client timeout was not applied", elapsed)
	}
}

// TestNoTimeoutlessHTTPClients guards the fix for
// https://github.com/konflux-ci/qe-tools/issues/221: production code must not
// reach for http.Get, http.DefaultClient or a bare &http.Client{}, all of which
// wait forever.
func TestNoTimeoutlessHTTPClients(t *testing.T) {
	forbidden := regexp.MustCompile(`http\.Get\(|http\.Head\(|http\.Post\(|http\.PostForm\(|http\.DefaultClient|&?http\.Client\{\}`)

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to resolve repository root: %v", err)
	}

	for _, dir := range []string{"cmd", "pkg"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			content, err := os.ReadFile(path) // #nosec G304 -- path comes from walking the repository itself
			if err != nil {
				return err
			}
			for i, line := range strings.Split(string(content), "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "//") {
					continue
				}
				if forbidden.MatchString(line) {
					rel, relErr := filepath.Rel(root, path)
					if relErr != nil {
						rel = path
					}
					t.Errorf("%s:%d uses an HTTP client without a timeout, use utils.NewHTTPClient(): %s",
						filepath.ToSlash(rel), i+1, strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to walk %s: %v", dir, err)
		}
	}
}

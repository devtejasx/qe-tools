package estimate

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/google/go-github/v56/github"
)

// newPaginatedServer serves `total` items for path, split into pages the way the
// GitHub API does it: honouring per_page and advertising the next page in a Link header.
func newPaginatedServer(t *testing.T, path string, total int, item func(i int) string) *github.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
		if perPage == 0 {
			perPage = 30
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page == 0 {
			page = 1
		}
		start := (page - 1) * perPage
		end := min(start+perPage, total)
		if end < total {
			next := *r.URL
			q := next.Query()
			q.Set("page", strconv.Itoa(page+1))
			next.RawQuery = q.Encode()
			w.Header().Set("Link", fmt.Sprintf(`<http://%s%s>; rel="next"`, r.Host, next.RequestURI()))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "[")
		for i := start; i < end; i++ {
			if i > start {
				fmt.Fprint(w, ",")
			}
			fmt.Fprint(w, item(i))
		}
		fmt.Fprint(w, "]")
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := github.NewClient(nil)
	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	client.BaseURL = baseURL
	return client
}

func TestGetChangedFilesReadsEveryPage(t *testing.T) {
	const total = 250
	client := newPaginatedServer(t, "/repos/o/r/pulls/1/files", total, func(i int) string {
		return fmt.Sprintf(`{"filename":"f%d.go","additions":1}`, i)
	})

	files, err := getChangedFiles(client, "o", "r", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != total {
		t.Fatalf("got %d files, want %d", len(files), total)
	}
	if got := files[total-1].GetFilename(); got != fmt.Sprintf("f%d.go", total-1) {
		t.Errorf("last file is %q", got)
	}
}

func TestCountCommitsReadsEveryPage(t *testing.T) {
	const total = 130
	client := newPaginatedServer(t, "/repos/o/r/pulls/1/commits", total, func(i int) string {
		return fmt.Sprintf(`{"sha":"%040d"}`, i)
	})

	count, err := countCommits(client, "o", "r", 1)
	if err != nil {
		t.Fatal(err)
	}
	if count != total {
		t.Fatalf("got %d commits, want %d", count, total)
	}
}

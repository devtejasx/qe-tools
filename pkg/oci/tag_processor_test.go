package oci

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"oras.land/oras-go/v2/registry/remote"
)

func TestOCICopyStopsAtContextDeadline(t *testing.T) {
	// A registry that accepts the request and then never answers.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	repo, err := remote.NewRepository(strings.TrimPrefix(server.URL, "http://") + "/org/artifacts")
	if err != nil {
		t.Fatal(err)
	}
	repo.PlainHTTP = true

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- (&Controller{}).ociCopy(ctx, repo, "latest", t.TempDir())
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error from a registry that never answers")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected the context deadline to stop the copy, got: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ociCopy ignored the context deadline")
	}
}

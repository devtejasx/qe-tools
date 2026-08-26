package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHTTPClientHasExplicitTimeout(t *testing.T) {
	client := NewHTTPClient()

	if client == nil {
		t.Fatal("NewHTTPClient returned nil")
	}
	if client.Timeout == 0 {
		t.Error("NewHTTPClient returned a client with no timeout, which can hang indefinitely")
	}
	if client.Timeout != DefaultHTTPTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultHTTPTimeout, client.Timeout)
	}
	if client == http.DefaultClient {
		t.Error("NewHTTPClient must not return the shared http.DefaultClient")
	}
}

func TestNewHTTPClientReturnsIndependentClients(t *testing.T) {
	a, b := NewHTTPClient(), NewHTTPClient()
	if a == b {
		t.Error("NewHTTPClient must return a fresh client so callers cannot mutate a shared one")
	}
}

func TestNewHTTPClientPerformsRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	resp, err := NewHTTPClient().Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func TestNewHTTPClientTimesOutOnSlowServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.Timeout = 20 * time.Millisecond

	if _, err := client.Get(server.URL); err == nil {
		t.Error("expected the request to fail once the client timeout elapsed")
	}
}

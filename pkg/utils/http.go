package utils

import (
	"net/http"
	"time"
)

// DefaultHTTPTimeout bounds every outbound HTTP request made by qe-tools.
// The Go standard library's http.DefaultClient has no timeout at all, so a
// stalled connection hangs a command forever instead of failing it.
const DefaultHTTPTimeout = 30 * time.Second

// NewHTTPClient returns an http.Client with an explicit timeout. Prefer it over
// http.Get, http.DefaultClient or a bare &http.Client{}, none of which time out.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: DefaultHTTPTimeout}
}

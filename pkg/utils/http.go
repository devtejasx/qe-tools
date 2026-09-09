package utils

import (
	"net/http"
	"time"
)

// DefaultHTTPTimeout is the timeout applied to HTTP clients returned by
// NewHTTPClient. Go's http.DefaultClient has no timeout at all, so a request
// against an unresponsive host blocks the command forever.
const DefaultHTTPTimeout = 30 * time.Second

// NewHTTPClient returns an HTTP client with an explicit timeout. Use it instead
// of http.Get, http.DefaultClient or a bare &http.Client{}, none of which
// impose any deadline.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: DefaultHTTPTimeout}
}

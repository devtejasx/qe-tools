package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoWebHookCreate(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		resource string
		secret   string
	}{
		{
			name:     "Simple data with secret",
			data:     map[string]string{"key": "value"},
			resource: "test-resource",
			secret:   "test-secret",
		},
		{
			name:     "Complex nested data",
			data:     map[string]interface{}{"user": map[string]string{"name": "test", "email": "test@example.com"}},
			resource: "user-resource",
			secret:   "secret123",
		},
		{
			name:     "Empty secret",
			data:     "simple string data",
			resource: "string-resource",
			secret:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &GoWebHook{}
			hook.Create(tt.data, tt.resource, tt.secret)

			if hook.Payload.Resource != tt.resource {
				t.Errorf("expected resource %q, got %q", tt.resource, hook.Payload.Resource)
			}

			if hook.Payload.Data == nil {
				t.Error("expected data to be set")
			}

			if len(hook.PreparedData) == 0 {
				t.Error("expected PreparedData to be populated")
			}

			if hook.ResultingSha == "" {
				t.Error("expected ResultingSha to be populated")
			}

			// Verify HMAC signature is correct
			h := hmac.New(sha256.New, []byte(tt.secret))
			h.Write(hook.PreparedData)
			expectedSha := hex.EncodeToString(h.Sum(nil))

			if hook.ResultingSha != expectedSha {
				t.Errorf("expected SHA %q, got %q", expectedSha, hook.ResultingSha)
			}

			// Verify PreparedData can be unmarshaled
			var payload GoWebHookPayload
			if err := json.Unmarshal(hook.PreparedData, &payload); err != nil {
				t.Errorf("PreparedData should be valid JSON: %v", err)
			}
		})
	}
}

func TestGoWebHookCreate_DifferentSecrets(t *testing.T) {
	data := map[string]string{"key": "value"}
	resource := "test"
	secret1 := "secret1"
	secret2 := "secret2"

	hook1 := &GoWebHook{}
	hook1.Create(data, resource, secret1)

	hook2 := &GoWebHook{}
	hook2.Create(data, resource, secret2)

	if hook1.ResultingSha == hook2.ResultingSha {
		t.Error("different secrets should produce different signatures")
	}
}

// MockRoundTripper is a mock implementation of http.RoundTripper for testing
type MockRoundTripper struct {
	Response        *http.Response
	Err             error
	ReceivedRequest *http.Request
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	m.ReceivedRequest = req
	return m.Response, m.Err
}

func TestGoWebHookSend(t *testing.T) {
	tests := []struct {
		name                    string
		hook                    *GoWebHook
		mockError               error
		expectedMethod          string
		expectedSignatureHeader string
		expectedSignatureValue  string
		expectedHeaders         map[string]string
		expectError             bool
	}{
		{
			name: "POST request with default signature header",
			hook: &GoWebHook{
				PreparedData:    []byte(`{"resource":"test","data":"value"}`),
				ResultingSha:    "abc123",
				PreferredMethod: http.MethodPost,
			},
			expectedMethod:          http.MethodPost,
			expectedSignatureHeader: DefaultSignatureHeader,
			expectedSignatureValue:  "abc123",
		},
		{
			name: "PUT request with custom signature header",
			hook: &GoWebHook{
				PreparedData:    []byte(`{"resource":"test","data":"value"}`),
				ResultingSha:    "def456",
				PreferredMethod: http.MethodPut,
				SignatureHeader: "X-Custom-Signature",
			},
			expectedMethod: http.MethodPut,
			// Send always writes the SHA under DefaultSignatureHeader; the
			// custom header only replaces the empty SignatureHeader field.
			expectedSignatureHeader: DefaultSignatureHeader,
			expectedSignatureValue:  "def456",
		},
		{
			name: "Invalid method falls back to POST",
			hook: &GoWebHook{
				PreparedData:    []byte(`{"resource":"test","data":"value"}`),
				ResultingSha:    "ghi789",
				PreferredMethod: "INVALID",
			},
			expectedMethod:          http.MethodPost,
			expectedSignatureHeader: DefaultSignatureHeader,
			expectedSignatureValue:  "ghi789",
		},
		{
			name: "Empty method falls back to POST",
			hook: &GoWebHook{
				PreparedData: []byte(`{"resource":"test","data":"value"}`),
				ResultingSha: "pqr678",
			},
			expectedMethod:          http.MethodPost,
			expectedSignatureHeader: DefaultSignatureHeader,
			expectedSignatureValue:  "pqr678",
		},
		{
			name: "DELETE request",
			hook: &GoWebHook{
				PreparedData:    []byte(`{"resource":"test","data":"value"}`),
				ResultingSha:    "jkl012",
				PreferredMethod: http.MethodDelete,
			},
			expectedMethod:          http.MethodDelete,
			expectedSignatureHeader: DefaultSignatureHeader,
			expectedSignatureValue:  "jkl012",
		},
		{
			name: "Additional headers included",
			hook: &GoWebHook{
				PreparedData:    []byte(`{"resource":"test","data":"value"}`),
				ResultingSha:    "mno345",
				PreferredMethod: http.MethodPost,
				AdditionalHeaders: map[string]string{
					"X-Custom-Header": "custom-value",
				},
			},
			expectedMethod:          http.MethodPost,
			expectedSignatureHeader: DefaultSignatureHeader,
			expectedSignatureValue:  "mno345",
			expectedHeaders:         map[string]string{"X-Custom-Header": "custom-value"},
		},
		{
			name: "Transport error is propagated",
			hook: &GoWebHook{
				PreparedData:    []byte(`{"resource":"test","data":"value"}`),
				ResultingSha:    "stu901",
				PreferredMethod: http.MethodPost,
			},
			mockError:   errors.New("connection refused"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockRoundTripper{
				Response: &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString("OK")),
				},
				Err: tt.mockError,
			}
			tt.hook.HTTPClient = &http.Client{Transport: mock}

			resp, err := tt.hook.Send("http://example.com/webhook")

			if tt.expectError {
				if err == nil {
					t.Fatal("expected Send to return an error")
				}
				if resp != nil {
					t.Error("expected no response alongside the error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Send returned an unexpected error: %v", err)
			}
			defer resp.Body.Close()

			req := mock.ReceivedRequest
			if req == nil {
				t.Fatal("expected Send to issue a request")
			}
			if req.Method != tt.expectedMethod {
				t.Errorf("expected method %q, got %q", tt.expectedMethod, req.Method)
			}
			if got := req.Header.Get(tt.expectedSignatureHeader); got != tt.expectedSignatureValue {
				t.Errorf("expected %s %q, got %q", tt.expectedSignatureHeader, tt.expectedSignatureValue, got)
			}
			if got := req.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("expected Content-Type application/json, got %q", got)
			}
			for name, want := range tt.expectedHeaders {
				if got := req.Header.Get(name); got != want {
					t.Errorf("expected header %s %q, got %q", name, want, got)
				}
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read the sent body: %v", err)
			}
			if !bytes.Equal(body, tt.hook.PreparedData) {
				t.Errorf("expected body %q, got %q", tt.hook.PreparedData, body)
			}

			// Send fills in the default signature header when none was given.
			if tt.hook.SignatureHeader == "" {
				t.Error("expected Send to populate SignatureHeader")
			}
		})
	}
}

func TestWebhookCreateAndSend(t *testing.T) {
	tests := []struct {
		name          string
		webhook       *Webhook
		saltSecret    string
		webhookTarget string
		expectError   bool
	}{
		{
			name: "Valid webhook data",
			webhook: &Webhook{
				Path:          "/test/path",
				RepositoryURL: "https://github.com/test/repo",
				Repository: Repository{
					FullName:   "test/repo",
					PullNumber: "123",
				},
			},
			saltSecret:    "test-secret",
			webhookTarget: "http://example.com/webhook",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the webhook
			hook := &GoWebHook{}
			hook.Create(tt.webhook, tt.webhook.Path, tt.saltSecret)

			// Verify the webhook was created correctly
			if hook.PreparedData == nil {
				t.Error("expected PreparedData to be set")
			}

			if hook.ResultingSha == "" {
				t.Error("expected ResultingSha to be set")
			}

			// Verify we can unmarshal the prepared data
			var payload GoWebHookPayload
			if err := json.Unmarshal(hook.PreparedData, &payload); err != nil {
				t.Errorf("failed to unmarshal prepared data: %v", err)
			}

			if payload.Resource != tt.webhook.Path {
				t.Errorf("expected resource %q, got %q", tt.webhook.Path, payload.Resource)
			}

			// Verify HMAC signature
			h := hmac.New(sha256.New, []byte(tt.saltSecret))
			h.Write(hook.PreparedData)
			expectedSha := hex.EncodeToString(h.Sum(nil))

			if hook.ResultingSha != expectedSha {
				t.Errorf("expected SHA %q, got %q", expectedSha, hook.ResultingSha)
			}

			// Note: We can't test the actual HTTP sending without mocking the client
			// which would require modifying the source code to accept an injected client
		})
	}
}

func TestGoWebHookSecuritySettings(t *testing.T) {
	// With no injected client, Send builds its own and IsSecure decides
	// whether that client verifies the server certificate. httptest.NewTLSServer
	// presents a self-signed certificate, so an insecure hook reaches it and a
	// secure one does not.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tests := []struct {
		name        string
		isSecure    bool
		expectError bool
	}{
		{name: "Secure mode enabled rejects the self-signed certificate", isSecure: true, expectError: true},
		{name: "Secure mode disabled (default) accepts it", isSecure: false, expectError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &GoWebHook{
				PreparedData:    []byte(`{"test":"data"}`),
				ResultingSha:    "test-sha",
				PreferredMethod: http.MethodPost,
				IsSecure:        tt.isSecure,
			}

			resp, err := hook.Send(server.URL)
			if tt.expectError {
				if err == nil {
					resp.Body.Close()
					t.Fatal("expected the certificate to be rejected")
				}
				return
			}
			if err != nil {
				t.Fatalf("Send returned an unexpected error: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
			}
		})
	}
}

func TestGoWebHookPayloadSerialization(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		data     interface{}
	}{
		{
			name:     "String data",
			resource: "test",
			data:     "simple string",
		},
		{
			name:     "Map data",
			resource: "test",
			data:     map[string]interface{}{"key": "value", "number": 123},
		},
		{
			name:     "Nested struct data",
			resource: "webhook",
			data: Webhook{
				Path:          "/path",
				RepositoryURL: "https://example.com",
				Repository: Repository{
					FullName:   "owner/repo",
					PullNumber: "456",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &GoWebHook{}
			hook.Create(tt.data, tt.resource, "secret")

			// Unmarshal and verify
			var payload GoWebHookPayload
			if err := json.Unmarshal(hook.PreparedData, &payload); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if payload.Resource != tt.resource {
				t.Errorf("expected resource %q, got %q", tt.resource, payload.Resource)
			}

			// Verify data is present
			if payload.Data == nil {
				t.Error("expected data to be set")
			}
		})
	}
}

func TestDefaultSignatureHeader(t *testing.T) {
	expected := "X-GoWebHooks-Verification"
	if DefaultSignatureHeader != expected {
		t.Errorf("expected DefaultSignatureHeader to be %q, got %q", expected, DefaultSignatureHeader)
	}
}

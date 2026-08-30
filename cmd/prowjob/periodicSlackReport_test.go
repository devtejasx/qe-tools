package prowjob

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchTextContent(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		responseStatus int
		wantErr        bool
		wantContains   string
	}{
		{
			name:           "successful fetch returns body",
			responseBody:   "hello world",
			responseStatus: http.StatusOK,
			wantContains:   "hello world",
		},
		{
			name:           "strips ANSI escape sequences",
			responseBody:   "\x1b[31mred text\x1b[0m",
			responseStatus: http.StatusOK,
			wantContains:   "red text",
		},
		{
			name:           "empty response body",
			responseBody:   "",
			responseStatus: http.StatusOK,
			wantContains:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			got, err := fetchTextContent(server.URL)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchTextContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantContains {
				t.Errorf("fetchTextContent() = %q, want %q", got, tt.wantContains)
			}
		})
	}
}

func TestFetchTextContentInvalidURL(t *testing.T) {
	_, err := fetchTextContent("http://invalid-host-that-does-not-exist.test:1")
	if err == nil {
		t.Error("fetchTextContent() expected error for invalid URL, got nil")
	}
}

func TestRemoveANSIEscapeSequences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no escape sequences",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "color codes removed",
			input: "\x1b[31mred\x1b[0m \x1b[32mgreen\x1b[0m",
			want:  "red green",
		},
		{
			name:  "bold and underline removed",
			input: "\x1b[1mbold\x1b[0m \x1b[4munderline\x1b[0m",
			want:  "bold underline",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeANSIEscapeSequences(tt.input)
			if got != tt.want {
				t.Errorf("removeANSIEscapeSequences() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsJobFailed(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "job failed",
			body: "some output\nReporting job state 'failed'\nmore output",
			want: true,
		},
		{
			name: "job succeeded",
			body: "some output\nReporting job state 'succeeded'\nmore output",
			want: false,
		},
		{
			name: "no state line",
			body: "no state info here",
			want: false,
		},
		{
			name: "empty body",
			body: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJobFailed(tt.body)
			if got != tt.want {
				t.Errorf("isJobFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractTestResultsAndSummary(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "valid test results",
			body: "Ran 10 of 20 Specs in 45.5 seconds\nFAIL! -- 8 Passed | 2 Failed | 0 Pending | 10 Skipped",
			want: "Test Results: 8 Passed | 2 Failed | 0 Pending | 10 Skipped\nRan 10 of 20 Specs in 45.5 seconds\n",
		},
		{
			name: "no matching pattern",
			body: "some random log output",
			want: "Infrastructure setup issues or failures unrelated to tests were found\n",
		},
		{
			name: "empty body",
			body: "",
			want: "Infrastructure setup issues or failures unrelated to tests were found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTestResultsAndSummary(tt.body)
			if got != tt.want {
				t.Errorf("extractTestResultsAndSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractDuration(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "duration found",
			body: "Ran for 1h23m45s",
			want: "Total Duration: 1h23m45s\n",
		},
		{
			name: "no duration",
			body: "no duration here",
			want: "",
		},
		{
			name: "empty body",
			body: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractDuration(tt.body)
			if got != tt.want {
				t.Errorf("extractDuration() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatFailures(t *testing.T) {
	tests := []struct {
		name     string
		failures string
		want     string
	}{
		{
			name:     "multiple failures",
			failures: "Summarizing\n[FAIL] test one failed\nsome context\n[FAIL] test two failed\nTest Suite Failed",
			want:     "Failures:\n- [FAIL] test one failed\n- [FAIL] test two failed\n",
		},
		{
			name:     "no FAIL lines",
			failures: "Summarizing\nsome output\nTest Suite Failed",
			want:     "No specific failures captured in the report.\n",
		},
		{
			name:     "empty string",
			failures: "",
			want:     "No specific failures captured in the report.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFailures(tt.failures)
			if got != tt.want {
				t.Errorf("formatFailures() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConstructMessage(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantSuccess bool
	}{
		{
			name:        "successful job",
			body:        "Reporting job state 'succeeded'\nall good",
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, success := constructMessage(tt.body)
			if success != tt.wantSuccess {
				t.Errorf("constructMessage() success = %v, want %v", success, tt.wantSuccess)
			}
			if tt.wantSuccess && msg != "Job Succeeded" {
				t.Errorf("constructMessage() = %q, want 'Job Succeeded'", msg)
			}
		})
	}
}

func TestConstructMessageFailedWithoutSummaryBlock(t *testing.T) {
	// State is 'failed', but the log carries no "Summarizing ... Test Suite
	// Failed" block, so the failure regexp does not match.
	body := "Reporting job state 'failed'\nRan for 3m12s\n"

	msg, success := constructMessage(body)
	if success {
		t.Fatalf("expected success=false for a failed job, got true")
	}
	if !strings.Contains(msg, "Test Suite Summary:") {
		t.Errorf("expected a summary header, got %q", msg)
	}
	if !strings.Contains(msg, "No specific failures captured in the report.") {
		t.Errorf("expected the no-failures fallback, got %q", msg)
	}
}

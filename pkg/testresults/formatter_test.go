package testresults

import (
	"strings"
	"testing"

	"github.com/bsm/ginkgo/v2/reporters"
)

func TestExtractFailedTestCasesBody(t *testing.T) {
	tests := []struct {
		name             string
		report           FailedTestCasesReport
		expectedLength   int
		expectedContains string
	}{
		{
			name: "OtherFailure type returns empty",
			report: FailedTestCasesReport{
				FailureType: OtherFailure,
			},
			expectedLength: 0,
		},
		{
			name: "ClusterCreationFailure returns wrapped log",
			report: FailedTestCasesReport{
				FailureType:         ClusterCreationFailure,
				ClusterProvisionLog: "Cluster provision failed",
			},
			expectedLength:   1,
			expectedContains: "Cluster provision failed",
		},
		{
			name: "TestRunFailure returns wrapped E2E log",
			report: FailedTestCasesReport{
				FailureType: TestRunFailure,
				E2ETestLog:  "E2E test execution failed",
			},
			expectedLength:   1,
			expectedContains: "E2E test execution failed",
		},
		{
			name: "TestCaseFailure with failure message",
			report: FailedTestCasesReport{
				FailureType: TestCaseFailure,
				JUnitTestSuites: &reporters.JUnitTestSuites{
					TestSuites: []reporters.JUnitTestSuite{
						{
							Name:     "Test Suite",
							Failures: 1,
							TestCases: []reporters.JUnitTestCase{
								{
									Name:   "Failed Test",
									Status: "failed",
									Failure: &reporters.JUnitFailure{
										Message: "Assertion failed",
									},
								},
							},
						},
					},
				},
			},
			expectedLength:   1,
			expectedContains: "Failed Test",
		},
		{
			name: "TestCaseFailure with timedout status uses SystemErr",
			report: FailedTestCasesReport{
				FailureType: TestCaseFailure,
				JUnitTestSuites: &reporters.JUnitTestSuites{
					TestSuites: []reporters.JUnitTestSuite{
						{
							Name:     "Test Suite",
							Failures: 1,
							TestCases: []reporters.JUnitTestCase{
								{
									Name:      "Timed Out Test",
									Status:    "timedout",
									SystemErr: "Test exceeded timeout",
									Failure: &reporters.JUnitFailure{
										Message: "This should not be used",
									},
								},
							},
						},
					},
				},
			},
			expectedLength:   1,
			expectedContains: "Test exceeded timeout",
		},
		{
			name: "TestCaseFailure with error instead of failure",
			report: FailedTestCasesReport{
				FailureType: TestCaseFailure,
				JUnitTestSuites: &reporters.JUnitTestSuites{
					TestSuites: []reporters.JUnitTestSuite{
						{
							Name:   "Test Suite",
							Errors: 1,
							TestCases: []reporters.JUnitTestCase{
								{
									Name:   "Error Test",
									Status: "error",
									Error: &reporters.JUnitError{
										Message: "Runtime error occurred",
									},
								},
							},
						},
					},
				},
			},
			expectedLength:   1,
			expectedContains: "Runtime error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := extractFailedTestCasesBody(tt.report)

			if len(body) != tt.expectedLength {
				t.Errorf("expected body length %d, got %d", tt.expectedLength, len(body))
			}

			if tt.expectedContains != "" && len(body) > 0 {
				found := false
				for _, entry := range body {
					if strings.Contains(entry, tt.expectedContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected body to contain %q, but it was not found in %v", tt.expectedContains, body)
				}
			}
		})
	}
}

func TestGetHeaderStringForFailureType(t *testing.T) {
	tests := []struct {
		name             string
		failureType      FailureType
		expectedContains string
	}{
		{
			name:             "OtherFailure header",
			failureType:      OtherFailure,
			expectedContains: "Couldn't detect a specific failure",
		},
		{
			name:             "TestRunFailure header",
			failureType:      TestRunFailure,
			expectedContains: "No JUnit file found",
		},
		{
			name:             "ClusterCreationFailure header",
			failureType:      ClusterCreationFailure,
			expectedContains: "Failed to provision a cluster",
		},
		{
			name:             "TestCaseFailure header",
			failureType:      TestCaseFailure,
			expectedContains: "Error occurred while running the E2E tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := getHeaderStringForFailureType(tt.failureType)

			if !strings.Contains(header, tt.expectedContains) {
				t.Errorf("expected header to contain %q, got %q", tt.expectedContains, header)
			}

			if !strings.Contains(header, ":rotating_light:") {
				t.Errorf("expected header to contain emoji :rotating_light:, got %q", header)
			}
		})
	}
}

func TestGetFormattedReport(t *testing.T) {
	tests := []struct {
		name             string
		report           FailedTestCasesReport
		expectedContains []string
	}{
		{
			name: "OtherFailure report",
			report: FailedTestCasesReport{
				FailureType: OtherFailure,
			},
			expectedContains: []string{
				":rotating_light:",
				"Couldn't detect a specific failure",
			},
		},
		{
			name: "TestCaseFailure with multiple failures",
			report: FailedTestCasesReport{
				FailureType: TestCaseFailure,
				JUnitTestSuites: &reporters.JUnitTestSuites{
					TestSuites: []reporters.JUnitTestSuite{
						{
							Name:     "Test Suite",
							Failures: 2,
							TestCases: []reporters.JUnitTestCase{
								{
									Name:   "Failed Test 1",
									Status: "failed",
									Failure: &reporters.JUnitFailure{
										Message: "First failure",
									},
								},
								{
									Name:   "Failed Test 2",
									Status: "failed",
									Failure: &reporters.JUnitFailure{
										Message: "Second failure",
									},
								},
							},
						},
					},
				},
			},
			expectedContains: []string{
				"Error occurred while running the E2E tests",
				"Failed Test 1",
				"Failed Test 2",
			},
		},
		{
			name: "ClusterCreationFailure report",
			report: FailedTestCasesReport{
				FailureType:         ClusterCreationFailure,
				ClusterProvisionLog: "Failed to create cluster",
			},
			expectedContains: []string{
				"Failed to provision a cluster",
				"Failed to create cluster",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formattedReport := GetFormattedReport(tt.report)

			for _, expected := range tt.expectedContains {
				if !strings.Contains(formattedReport, expected) {
					t.Errorf("expected report to contain %q, but it was not found", expected)
				}
			}
		})
	}
}

func TestReturnTruncatedContent(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		expectedTruncate bool
		expectedLength   int
	}{
		{
			name:             "Content under 10000 chars",
			content:          strings.Repeat("a", 9999),
			expectedTruncate: false,
			expectedLength:   9999,
		},
		{
			name:             "Content exactly 10000 chars",
			content:          strings.Repeat("b", 10000),
			expectedTruncate: false,
			expectedLength:   10000,
		},
		{
			name:             "Content over 10000 chars",
			content:          strings.Repeat("c", 15000),
			expectedTruncate: true,
			expectedLength:   10000,
		},
		{
			name:             "Empty content",
			content:          "",
			expectedTruncate: false,
			expectedLength:   0,
		},
		{
			// 15000 bytes but only 5000 runes: was sliced past the end and panicked.
			name:             "Multi-byte content over 10000 bytes but under 10000 chars",
			content:          strings.Repeat("日", 5000),
			expectedTruncate: false,
			expectedLength:   5000,
		},
		{
			name:             "Multi-byte content over 10000 chars",
			content:          strings.Repeat("日", 10001),
			expectedTruncate: true,
			expectedLength:   10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := returnTruncatedContent(tt.content)

			if tt.expectedTruncate {
				if !strings.Contains(result, "the content is too long") {
					t.Error("expected truncation message in result")
				}
				// Count runes up to the truncation message
				runeCount := 0
				for _, r := range result {
					if r == '.' && runeCount >= tt.expectedLength {
						break
					}
					runeCount++
				}
				if runeCount < tt.expectedLength {
					t.Errorf("expected at least %d runes before truncation message, got %d", tt.expectedLength, runeCount)
				}
			} else {
				if result != tt.content {
					t.Errorf("expected result to equal input for non-truncated content")
				}
				if len([]rune(result)) != tt.expectedLength {
					t.Errorf("expected result length %d, got %d", tt.expectedLength, len([]rune(result)))
				}
			}
		})
	}
}

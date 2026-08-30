package prowjob

import (
	"testing"

	"github.com/konflux-ci/qe-tools/pkg/prow"
)

func TestBuildJUnitFromArtifacts_PassedStep(t *testing.T) {
	passed := true
	scanner := &prow.ArtifactScanner{
		ArtifactDirectoryPrefix: "logs/test-job/123/",
		ArtifactStepMap: map[prow.ArtifactStepName]prow.ArtifactFilenameMap{
			"e2e-tests": {
				"finished.json": prow.Artifact{
					Content:  `{"passed": true}`,
					FullName: "logs/test-job/123/e2e-tests/finished.json",
				},
				"build-log.txt": prow.Artifact{
					Content:  "test output logs",
					FullName: "logs/test-job/123/e2e-tests/build-log.txt",
				},
			},
		},
	}
	_ = passed

	suites, err := buildJUnitFromArtifacts(scanner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if suites.Tests != 1 {
		t.Errorf("expected 1 test, got %d", suites.Tests)
	}
	if suites.Failures != 0 {
		t.Errorf("expected 0 failures, got %d", suites.Failures)
	}

	found := false
	for _, suite := range suites.TestSuites {
		if suite.Name == openshiftCITestSuiteName {
			for _, tc := range suite.TestCases {
				if tc.Name == "e2e-tests" {
					found = true
					if tc.Status != "passed" {
						t.Errorf("expected status 'passed', got %q", tc.Status)
					}
					if tc.SystemErr != "" {
						t.Error("expected SystemErr to be cleared for passed tests")
					}
				}
			}
		}
	}
	if !found {
		t.Error("test case 'e2e-tests' not found in openshift-ci suite")
	}
}

func TestBuildJUnitFromArtifacts_FailedStep(t *testing.T) {
	scanner := &prow.ArtifactScanner{
		ArtifactDirectoryPrefix: "logs/test-job/456/",
		ArtifactStepMap: map[prow.ArtifactStepName]prow.ArtifactFilenameMap{
			"integration-tests": {
				"finished.json": prow.Artifact{
					Content:  `{"passed": false}`,
					FullName: "logs/test-job/456/integration-tests/finished.json",
				},
				"build-log.txt": prow.Artifact{
					Content:  "FAIL: some test failed",
					FullName: "logs/test-job/456/integration-tests/build-log.txt",
				},
			},
		},
	}

	suites, err := buildJUnitFromArtifacts(scanner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if suites.Tests != 1 {
		t.Errorf("expected 1 test, got %d", suites.Tests)
	}
	if suites.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", suites.Failures)
	}

	for _, suite := range suites.TestSuites {
		if suite.Name == openshiftCITestSuiteName {
			for _, tc := range suite.TestCases {
				if tc.Name == "integration-tests" {
					if tc.Status != "failed" {
						t.Errorf("expected status 'failed', got %q", tc.Status)
					}
					if tc.Failure == nil {
						t.Error("expected Failure to be set")
					}
					if tc.SystemErr != "FAIL: some test failed" {
						t.Errorf("expected build log in SystemErr, got %q", tc.SystemErr)
					}
				}
			}
		}
	}
}

func TestBuildJUnitFromArtifacts_GatherStep(t *testing.T) {
	scanner := &prow.ArtifactScanner{
		ArtifactDirectoryPrefix: "logs/test-job/789/",
		ArtifactStepMap: map[prow.ArtifactStepName]prow.ArtifactFilenameMap{
			"gather-extra": {
				"finished.json": prow.Artifact{
					Content:  `{"passed": true}`,
					FullName: "logs/test-job/789/gather-extra/finished.json",
				},
			},
		},
	}

	suites, err := buildJUnitFromArtifacts(scanner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, suite := range suites.TestSuites {
		if suite.Name == openshiftCITestSuiteName {
			for _, prop := range suite.Properties.Properties {
				if prop.Name == "gather-extra" {
					found = true
					expected := gcsBrowserURLPrefix + "logs/test-job/789/gather-extra/artifacts"
					if prop.Value != expected {
						t.Errorf("expected property value %q, got %q", expected, prop.Value)
					}
				}
			}
		}
	}
	if !found {
		t.Error("gather-extra property not found")
	}
}

func TestBuildJUnitFromArtifacts_InvalidFinishedJSON(t *testing.T) {
	scanner := &prow.ArtifactScanner{
		ArtifactDirectoryPrefix: "logs/test-job/000/",
		ArtifactStepMap: map[prow.ArtifactStepName]prow.ArtifactFilenameMap{
			"broken-step": {
				"finished.json": prow.Artifact{
					Content:  `{invalid json`,
					FullName: "logs/test-job/000/broken-step/finished.json",
				},
			},
		},
	}

	_, err := buildJUnitFromArtifacts(scanner)
	if err == nil {
		t.Error("expected error for invalid finished.json")
	}
}

func TestBuildJUnitFromArtifacts_WithJUnitXML(t *testing.T) {
	junitXML := `<?xml version="1.0" encoding="UTF-8"?>
<testsuites tests="2" failures="0">
  <testsuite name="my-tests" tests="2" failures="0" timestamp="2024-01-15T10:00:00">
    <testcase name="test1" status="passed"/>
    <testcase name="test2" status="passed"/>
  </testsuite>
</testsuites>`

	scanner := &prow.ArtifactScanner{
		ArtifactDirectoryPrefix: "logs/test-job/111/",
		ArtifactStepMap: map[prow.ArtifactStepName]prow.ArtifactFilenameMap{
			"e2e-tests": {
				"finished.json": prow.Artifact{
					Content:  `{"passed": true}`,
					FullName: "logs/test-job/111/e2e-tests/finished.json",
				},
				"junit.xml": prow.Artifact{
					Content:  junitXML,
					FullName: "logs/test-job/111/e2e-tests/junit.xml",
				},
			},
		},
	}

	suites, err := buildJUnitFromArtifacts(scanner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(suites.TestSuites) < 2 {
		t.Fatalf("expected at least 2 test suites (parsed + openshift-ci), got %d", len(suites.TestSuites))
	}
}

func TestBuildJUnitFromArtifacts_EmptyArtifacts(t *testing.T) {
	scanner := &prow.ArtifactScanner{
		ArtifactDirectoryPrefix: "logs/test-job/222/",
		ArtifactStepMap:         map[prow.ArtifactStepName]prow.ArtifactFilenameMap{},
	}

	suites, err := buildJUnitFromArtifacts(scanner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if suites.Tests != 0 {
		t.Errorf("expected 0 tests for empty artifacts, got %d", suites.Tests)
	}
	if len(suites.TestSuites) != 1 {
		t.Errorf("expected 1 suite (openshift-ci), got %d", len(suites.TestSuites))
	}
}

func TestBuildJUnitFromArtifacts_MissingPassedKey(t *testing.T) {
	// finished.json without a "passed" key leaves Finished.Passed nil.
	scanner := &prow.ArtifactScanner{
		ArtifactDirectoryPrefix: "logs/test-job/123/",
		ArtifactStepMap: map[prow.ArtifactStepName]prow.ArtifactFilenameMap{
			"e2e-tests": {
				"finished.json": prow.Artifact{
					Content:  `{"timestamp": 1700000000}`,
					FullName: "logs/test-job/123/e2e-tests/finished.json",
				},
			},
		},
	}

	suites, err := buildJUnitFromArtifacts(scanner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if suites.Tests != 1 {
		t.Errorf("expected 1 test, got %d", suites.Tests)
	}
	if suites.Failures != 1 {
		t.Errorf("expected the step with no verdict to count as a failure, got %d", suites.Failures)
	}
}

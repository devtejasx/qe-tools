package utils

import (
	"strings"
	"testing"
)

func TestParseRepoAndTag(t *testing.T) {
	tests := []struct {
		name          string
		repoFlag      string
		expectedRepo  string
		expectedTag   string
		expectedError string
	}{
		{
			name:         "Valid format with simple repo and tag",
			repoFlag:     "quay.io/repo:tag",
			expectedRepo: "repo",
			expectedTag:  "tag",
		},
		{
			name:         "Valid format with nested repo",
			repoFlag:     "quay.io/org/repo:v1.0",
			expectedRepo: "org/repo",
			expectedTag:  "v1.0",
		},
		{
			name:         "Multiple colons - splits on first colon",
			repoFlag:     "quay.io/repo:tag:extra",
			expectedRepo: "repo",
			expectedTag:  "tag:extra",
		},
		{
			name:          "Missing quay.io prefix",
			repoFlag:      "docker.io/repo:tag",
			expectedError: "must start with 'quay.io/'",
		},
		{
			name:          "Missing tag",
			repoFlag:      "quay.io/repo",
			expectedError: "tag is missing",
		},
		{
			name:          "Empty string",
			repoFlag:      "",
			expectedError: "must start with 'quay.io/'",
		},
		{
			name:          "Only quay.io prefix",
			repoFlag:      "quay.io/",
			expectedError: "tag is missing",
		},
		{
			name:          "Empty tag",
			repoFlag:      "quay.io/org/repo:",
			expectedError: "tag is missing",
		},
		{
			name:          "Empty repository",
			repoFlag:      "quay.io/:tag",
			expectedError: "repository is missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, tag, err := ParseRepoAndTag(tt.repoFlag)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.expectedError)
					return
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if repo != tt.expectedRepo {
				t.Errorf("expected repo %q, got %q", tt.expectedRepo, repo)
			}

			if tag != tt.expectedTag {
				t.Errorf("expected tag %q, got %q", tt.expectedTag, tag)
			}
		})
	}
}

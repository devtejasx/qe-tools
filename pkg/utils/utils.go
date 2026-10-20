package utils

import (
	"fmt"
	"strings"
)

// ParseRepoAndTag extracts the quay.io repository and tag from the given repo flag.
func ParseRepoAndTag(repoFlag string) (string, string, error) {
	// Ensure the repoFlag starts with 'quay.io/'
	if !strings.HasPrefix(repoFlag, "quay.io/") {
		return "", "", fmt.Errorf("the repository must start with 'quay.io/'")
	}

	// Remove 'quay.io/' prefix and split the repo and tag using the ':' character
	repoFlag = strings.TrimPrefix(repoFlag, "quay.io/")
	parts := strings.SplitN(repoFlag, ":", 2)

	if len(parts) != 2 || parts[1] == "" {
		return "", "", fmt.Errorf("tag is missing in the repo flag")
	}
	if parts[0] == "" {
		return "", "", fmt.Errorf("repository is missing in the repo flag")
	}

	return parts[0], parts[1], nil
}

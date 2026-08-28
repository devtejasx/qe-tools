package prow

import (
	"time"

	"cloud.google.com/go/storage"
)

const (
	// The name of the openshift-ci step where the "createReport" command is used
	reportStepName    = "redhat-appstudio-report"
	bucketName        = "test-platform-results"
	prowJobYAMLPrefix = "https://prow.ci.openshift.org/prowjob?prowjob="

	// httpClientTimeout bounds every request made from this package; the
	// default http.Client has no timeout at all.
	httpClientTimeout = 30 * time.Second
)

// ArtifactScanner is used for initializing
// GCS client and scanning and storing
// files found in defined storage
type ArtifactScanner struct {
	bucketHandle *storage.BucketHandle
	Client       *storage.Client
	config       ScannerConfig
	/* Example:
	{
	  "gather-extra": {"build-log.txt": {Content: "<content>", FullName: "/full/gcs/path/build-log.txt"}, "finished.json": ...},
	  "e2e-tests": {"build-log.txt": ...},
	}
	*/
	ArtifactStepMap         map[ArtifactStepName]ArtifactFilenameMap
	ArtifactDirectoryPrefix string
}

// ScannerConfig contains fields required
// for scaning files with ArtifactScanner
type ScannerConfig struct {
	FileNameFilter []string
	ProwJobID      string
	ProwJobURL     string
	StepsToSkip    []string
}

// ArtifactStepName represents the openshift-ci step name
type ArtifactStepName string

// ArtifactFilenameMap - e.g. "build-log.txt": {Content: "<file-content>", FullName: "/full/gcs/path/build-log.txt"}
type ArtifactFilenameMap map[ArtifactFilename]Artifact

// ArtifactFilename represents the name of the file (including file extension)
type ArtifactFilename string

// Artifact stores the full name of the artifact (in GCS) and the content of the file
type Artifact struct {
	Content  string
	FullName string
}

// OpenshiftJobSpec represents the Openshift job spec data
type OpenshiftJobSpec struct {
	Type string `json:"type"`
	Job  string `json:"job"`
	Refs Refs   `json:"refs"`
}

// Refs represent the refs field of an OpenShift job
type Refs struct {
	RepoLink     string `json:"repo_link"`
	Repo         string `json:"repo"`
	Organization string `json:"org"`
	Pulls        []Pull `json:"pulls"`
}

// Pull represents a GitHub Pull Request
type Pull struct {
	Number     int    `json:"number"`
	Author     string `json:"author"`
	SHA        string `json:"sha"`
	PRLink     string `json:"link"`
	AuthorLink string `json:"author_link"`
}

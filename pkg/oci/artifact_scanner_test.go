package oci

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessExtractedFiles(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"junit.xml":         "<testsuites/>",
		"e2e-tests.log":     "log",
		"unrelated.txt":     "ignored",
		"sub/cluster-1.log": "provision",
	}
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	as, err := NewArtifactScanner(ScannerConfig{FileNameFilter: []string{`junit\.xml$`, `\.log$`}})
	if err != nil {
		t.Fatal(err)
	}
	as.artifactDirPath = dir

	if err := as.processExtractedFiles(); err != nil {
		t.Fatalf("processExtractedFiles() error = %v", err)
	}

	if len(as.FilesPathMap) != 3 {
		t.Fatalf("got %d files, want 3: %v", len(as.FilesPathMap), as.FilesPathMap)
	}
	got := as.FilesPathMap[FilePath(filepath.Join(dir, "sub", "cluster-1.log"))]
	if got.Filename != "cluster-1.log" || got.Content != "provision" {
		t.Errorf("got %+v for sub/cluster-1.log", got)
	}

	// Every file read must have been closed again; on Windows an open
	// handle makes this fail.
	if err := os.RemoveAll(dir); err != nil {
		t.Errorf("cannot remove the scanned directory: %v", err)
	}
}

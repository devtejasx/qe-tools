package estimate

import (
	"testing"

	"github.com/google/go-github/v88/github"
)

func TestEstimateTimeToReviewCommand(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "estimate-review command has correct use",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if EstimateTimeToReviewCmd.Use != "estimate-review" {
				t.Errorf("expected Use to be 'estimate-review', got '%s'", EstimateTimeToReviewCmd.Use)
			}
			if EstimateTimeToReviewCmd.Short == "" {
				t.Error("expected Short description to be set")
			}
		})
	}
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "go file",
			filename: "main.go",
			want:     ".go",
		},
		{
			name:     "yaml file",
			filename: "config.yaml",
			want:     ".yaml",
		},
		{
			name:     "file with multiple dots",
			filename: "test.spec.ts",
			want:     ".ts",
		},
		{
			name:     "file without extension",
			filename: "README",
			want:     "",
		},
		{
			name:     "hidden file",
			filename: ".gitignore",
			want:     ".gitignore",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getFileExtension(tt.filename)
			if got != tt.want {
				t.Errorf("getFileExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEstimateFileTimes(t *testing.T) {
	tests := []struct {
		name  string
		files []*github.CommitFile
		want  int
	}{
		{
			name: "single file with additions",
			files: []*github.CommitFile{
				{
					Filename:  github.String("test.go"),
					Additions: github.Int(10),
					Deletions: github.Int(0),
				},
			},
			want: 20, // 10 * 2.0 (default) * 1.0 (base)
		},
		{
			name: "file with deletions",
			files: []*github.CommitFile{
				{
					Filename:  github.String("test.go"),
					Additions: github.Int(0),
					Deletions: github.Int(10),
				},
			},
			want: 10, // 10 * 2.0 (default) * 0.5 (deletion)
		},
		{
			name: "multiple files",
			files: []*github.CommitFile{
				{
					Filename:  github.String("test.go"),
					Additions: github.Int(5),
					Deletions: github.Int(5),
				},
				{
					Filename:  github.String("main.go"),
					Additions: github.Int(10),
					Deletions: github.Int(0),
				},
			},
			want: 35, // (5*2*1 + 5*2*0.5) + (10*2*1)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateFileTimes(tt.files)
			if got != tt.want {
				t.Errorf("estimateFileTimes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLabelBasedOnTime(t *testing.T) {
	// Set up test config
	config.Labels = []TimeLabel{
		{Name: "size/XS", Time: 0},
		{Name: "size/S", Time: 300},
		{Name: "size/M", Time: 900},
		{Name: "size/L", Time: 1800},
	}

	tests := []struct {
		name       string
		reviewTime int
		want       string
		wantErr    bool
	}{
		{
			name:       "very small PR",
			reviewTime: 100,
			want:       "size/XS",
			wantErr:    false,
		},
		{
			name:       "small PR",
			reviewTime: 500,
			want:       "size/S",
			wantErr:    false,
		},
		{
			name:       "medium PR",
			reviewTime: 1000,
			want:       "size/M",
			wantErr:    false,
		},
		{
			name:       "large PR",
			reviewTime: 2000,
			want:       "size/L",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getLabelBasedOnTime(tt.reviewTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("getLabelBasedOnTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Name != tt.want {
				t.Errorf("getLabelBasedOnTime() = %v, want %v", got.Name, tt.want)
			}
		})
	}
}

func TestGetLabelBasedOnTime_EmptyLabels(t *testing.T) {
	// Save original config
	originalLabels := config.Labels
	defer func() {
		config.Labels = originalLabels
	}()

	// Set empty labels
	config.Labels = []TimeLabel{}

	_, err := getLabelBasedOnTime(100)
	if err != errEmptyLabels {
		t.Errorf("expected errEmptyLabels, got %v", err)
	}
}

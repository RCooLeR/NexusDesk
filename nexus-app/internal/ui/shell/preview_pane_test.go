package shell

import (
	"testing"

	"nexusdesk/internal/domain"
)

func TestIsMarkdownPreview(t *testing.T) {
	cases := []struct {
		name    string
		preview domain.FilePreview
		want    bool
	}{
		{name: "md extension", preview: domain.FilePreview{RelPath: "docs/readme.md"}, want: true},
		{name: "markdown media type", preview: domain.FilePreview{RelPath: "readme.txt", MediaType: "text/markdown"}, want: true},
		{name: "plain text", preview: domain.FilePreview{RelPath: "main.go", MediaType: "text/plain"}, want: false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMarkdownPreview(tt.preview); got != tt.want {
				t.Fatalf("isMarkdownPreview() = %v, want %v", got, tt.want)
			}
		})
	}
}

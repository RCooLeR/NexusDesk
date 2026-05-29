package editor

import (
	"fmt"
	"testing"
)

func TestBuildBreadcrumbsKeepsIDEBehavior(t *testing.T) {
	tests := []struct {
		name          string
		activeFile    string
		workspaceName string
		want          []string
	}{
		{
			name:          "root only",
			activeFile:    "",
			workspaceName: "NexusDesk",
			want:          []string{"NexusDesk="},
		},
		{
			name:          "normalizes slashes",
			activeFile:    `docs\guide\README.md`,
			workspaceName: "Workspace",
			want:          []string{"Workspace=", "docs=docs", "guide=docs/guide", "README.md=docs/guide/README.md"},
		},
		{
			name:          "trims edge slashes",
			activeFile:    "/src/app.go/",
			workspaceName: "Code",
			want:          []string{"Code=", "src=src", "app.go=src/app.go"},
		},
		{
			name:          "defaults workspace label",
			activeFile:    "main.go",
			workspaceName: "",
			want:          []string{"Workspace=", "main.go=main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := breadcrumbSignatures(BuildBreadcrumbs(tt.activeFile, tt.workspaceName))
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Fatalf("unexpected breadcrumbs\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func breadcrumbSignatures(items []Breadcrumb) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Label+"="+item.RelPath)
	}
	return out
}

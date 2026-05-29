package editor

import (
	"fmt"
	"testing"
)

func TestBuildOutlineKeepsIDESymbolRules(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		content  string
		want     []string
	}{
		{
			name:     "markdown headings",
			fileName: "README.md",
			content:  "# Title\n\n## Setup\n### Details\n",
			want:     []string{"heading:Title:1:0", "heading:Setup:3:1", "heading:Details:4:2"},
		},
		{
			name:     "go functions and types",
			fileName: "main.go",
			content:  "package main\n\ntype Server struct{}\nfunc (s *Server) Run() {}\nfunc helper() {}\n",
			want:     []string{"type:Server:3:0", "func:Run:4:0", "func:helper:5:0"},
		},
		{
			name:     "typescript declarations",
			fileName: "app.tsx",
			content:  "export interface Props {}\nexport class Shell {}\nconst render = () => null\nexport async function load() {}\n",
			want:     []string{"interface:Props:1:0", "class:Shell:2:0", "func:render:3:0", "func:load:4:0"},
		},
		{
			name:     "css selectors",
			fileName: "style.scss",
			content:  ".app-shell {\n  color: red;\n}\n#root {\n}\n",
			want:     []string{"selector:.app-shell:1:0", "selector:#root:4:0"},
		},
		{
			name:     "json shallow keys",
			fileName: "config.json",
			content:  "{\n  \"name\": \"nexus\",\n    \"scripts\": {},\n          \"tooDeep\": true\n}\n",
			want:     []string{"key:name:2:1", "key:scripts:3:2"},
		},
		{
			name:     "yaml shallow keys",
			fileName: "compose.yml",
			content:  "services:\n  app:\n    image: nexus\n          tooDeep: true\n",
			want:     []string{"key:services:1:0", "key:app:2:1", "key:image:3:2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := outlineSignatures(BuildOutline(tt.fileName, tt.content))
			if fmt.Sprint(got) != fmt.Sprint(tt.want) {
				t.Fatalf("unexpected outline\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func TestBuildOutlineCapsLargeFiles(t *testing.T) {
	content := ""
	for i := 0; i < outlineMaxItems+20; i++ {
		content += fmt.Sprintf("# Heading %d\n", i)
	}

	got := BuildOutline("README.md", content)
	if len(got) != outlineMaxItems {
		t.Fatalf("expected %d capped outline items, got %d", outlineMaxItems, len(got))
	}
}

func outlineSignatures(items []OutlineItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("%s:%s:%d:%d", item.Kind, item.Label, item.Line, item.Level))
	}
	return out
}

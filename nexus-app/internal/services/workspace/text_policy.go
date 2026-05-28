package workspace

import (
	"path/filepath"
	"strings"
)

var textLikeExtensions = map[string]struct{}{
	".c":          {},
	".conf":       {},
	".cpp":        {},
	".cs":         {},
	".css":        {},
	".env":        {},
	".go":         {},
	".h":          {},
	".hpp":        {},
	".html":       {},
	".ini":        {},
	".java":       {},
	".js":         {},
	".json":       {},
	".jsonl":      {},
	".jsx":        {},
	".less":       {},
	".log":        {},
	".lock":       {},
	".md":         {},
	".ndjson":     {},
	".php":        {},
	".ps1":        {},
	".py":         {},
	".rb":         {},
	".rs":         {},
	".rtf":        {},
	".sass":       {},
	".scss":       {},
	".sh":         {},
	".sql":        {},
	".svelte":     {},
	".toml":       {},
	".ts":         {},
	".tsx":        {},
	".txt":        {},
	".vue":        {},
	".xml":        {},
	".yaml":       {},
	".yml":        {},
	".dockerfile": {},
}

var textLikeBasenames = map[string]struct{}{
	".dockerignore":  {},
	".editorconfig":  {},
	".env":           {},
	".gitattributes": {},
	".gitignore":     {},
	".npmrc":         {},
	"dockerfile":     {},
	"gemfile":        {},
	"go.mod":         {},
	"go.sum":         {},
	"makefile":       {},
	"procfile":       {},
}

func isTextLikePath(relPath string) bool {
	base := strings.ToLower(filepath.Base(filepath.FromSlash(relPath)))
	if _, ok := textLikeBasenames[base]; ok {
		return true
	}
	extension := strings.ToLower(filepath.Ext(relPath))
	_, ok := textLikeExtensions[extension]
	return ok
}

package editor

import (
	"path"
	"strings"
)

type Breadcrumb struct {
	Label   string
	RelPath string
}

func BuildBreadcrumbs(activeFile string, workspaceName string) []Breadcrumb {
	workspaceName = strings.TrimSpace(workspaceName)
	if workspaceName == "" {
		workspaceName = "Workspace"
	}
	normalized := normalizeBreadcrumbRelPath(activeFile)
	crumbs := []Breadcrumb{{Label: workspaceName, RelPath: ""}}
	if normalized == "" {
		return crumbs
	}
	parts := strings.Split(normalized, "/")
	for index, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		crumbs = append(crumbs, Breadcrumb{
			Label:   part,
			RelPath: strings.Join(parts[:index+1], "/"),
		})
	}
	return crumbs
}

func normalizeBreadcrumbRelPath(value string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	normalized = strings.Trim(normalized, "/")
	if normalized == "" {
		return ""
	}
	cleaned := path.Clean(normalized)
	if cleaned == "." {
		return ""
	}
	return strings.Trim(cleaned, "/")
}

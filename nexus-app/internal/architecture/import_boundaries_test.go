package architecture

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var forbiddenActiveAppImports = []string{
	"github.com/" + "w" + "ailsapp/",
	"github.com/" + "web" + "view/",
	"github.com/zserge/lorca",
	"web" + "view",
}

func TestActiveAppDoesNotImportDeprecatedRuntime(t *testing.T) {
	root := repoRoot(t)
	for _, file := range goFiles(t, root) {
		for _, importPath := range importsForFile(t, file) {
			if hasAnyPrefix(importPath, forbiddenActiveAppImports) {
				t.Fatalf("%s imports forbidden deprecated runtime dependency %q", rel(t, root, file), importPath)
			}
		}
	}
	assertGoModDoesNotReferenceForbiddenDesktopBridge(t, root)
}

func TestServicesAndDomainStayFrameworkFree(t *testing.T) {
	root := repoRoot(t)
	for _, file := range goFiles(t, filepath.Join(root, "internal")) {
		relative := filepath.ToSlash(rel(t, root, file))
		guarded := strings.HasPrefix(relative, "internal/services/") || strings.HasPrefix(relative, "internal/domain/")
		if !guarded {
			continue
		}
		for _, importPath := range importsForFile(t, file) {
			switch {
			case strings.HasPrefix(importPath, "fyne.io/fyne/"):
				t.Fatalf("%s imports Fyne %q; domain/services must stay framework-free", relative, importPath)
			case strings.HasPrefix(importPath, "nexusdesk/internal/ui"):
				t.Fatalf("%s imports UI package %q; dependency direction must stay UI -> services/domain", relative, importPath)
			}
		}
	}
}

func TestFyneImportsStayInDesktopPresentationPackages(t *testing.T) {
	root := repoRoot(t)
	for _, file := range goFiles(t, filepath.Join(root, "internal")) {
		relative := filepath.ToSlash(rel(t, root, file))
		for _, importPath := range importsForFile(t, file) {
			if !strings.HasPrefix(importPath, "fyne.io/fyne/") {
				continue
			}
			if allowedFyneImportPath(relative) {
				continue
			}
			t.Fatalf("%s imports Fyne %q outside approved app/brand/ui presentation packages", relative, importPath)
		}
	}
}

func allowedFyneImportPath(relative string) bool {
	return strings.HasPrefix(relative, "internal/app/") ||
		strings.HasPrefix(relative, "internal/brand/") ||
		strings.HasPrefix(relative, "internal/ui/")
}

func assertGoModDoesNotReferenceForbiddenDesktopBridge(t *testing.T, root string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	content := string(data)
	for _, forbidden := range forbiddenActiveAppImports {
		if strings.Contains(content, forbidden) {
			t.Fatalf("go.mod references forbidden deprecated runtime dependency %q", forbidden)
		}
	}
}

func importsForFile(t *testing.T, file string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse imports for %s: %v", file, err)
	}
	imports := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		imports = append(imports, strings.Trim(spec.Path.Value, `"`))
	}
	return imports
}

func goFiles(t *testing.T, root string) []string {
	t.Helper()
	files := []string{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "vendor" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return files
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod from %s", dir)
		}
		dir = parent
	}
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func rel(t *testing.T, root string, path string) string {
	t.Helper()
	relative, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatalf("make %s relative to %s: %v", path, root, err)
	}
	return relative
}

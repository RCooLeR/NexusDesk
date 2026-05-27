package operations

import (
	"path/filepath"
	"strings"
)

func classifyFile(relPath string, size int64) (File, bool) {
	name := filepath.Base(relPath)
	kind, ok := classifyKind(relPath, name)
	if !ok {
		return File{}, false
	}
	return File{
		RelPath: filepath.ToSlash(relPath),
		Name:    name,
		Kind:    kind,
		Size:    size,
	}, true
}

func classifyKind(relPath string, name string) (FileKind, bool) {
	lowerName := strings.ToLower(name)
	lowerPath := strings.ToLower(filepath.ToSlash(relPath))
	extension := strings.ToLower(filepath.Ext(lowerName))
	switch {
	case lowerName == "dockerfile" || strings.HasPrefix(lowerName, "dockerfile.") || extension == ".dockerfile":
		return FileKindDockerfile, true
	case isComposeFile(lowerName):
		return FileKindCompose, true
	case isEnvFile(lowerName, lowerPath):
		return FileKindEnv, true
	case extension == ".log" || strings.Contains(lowerPath, "/logs/") || strings.HasSuffix(lowerPath, "/logs"):
		return FileKindLog, true
	case isScriptExtension(extension):
		return FileKindScript, true
	case isConfigExtension(extension):
		return FileKindConfig, true
	default:
		return "", false
	}
}

func isComposeFile(lowerName string) bool {
	return lowerName == "compose.yml" ||
		lowerName == "compose.yaml" ||
		lowerName == "docker-compose.yml" ||
		lowerName == "docker-compose.yaml" ||
		(strings.Contains(lowerName, "compose") && (strings.HasSuffix(lowerName, ".yml") || strings.HasSuffix(lowerName, ".yaml")))
}

func isEnvFile(lowerName string, lowerPath string) bool {
	return lowerName == ".env" ||
		strings.HasPrefix(lowerName, ".env.") ||
		strings.HasSuffix(lowerName, ".env") ||
		strings.HasSuffix(lowerName, ".env.local") ||
		strings.HasSuffix(lowerName, ".env.production") ||
		strings.HasSuffix(lowerName, ".env.development") ||
		strings.Contains(lowerPath, "/.env")
}

func isScriptExtension(extension string) bool {
	switch extension {
	case ".bat", ".cmd", ".ps1", ".sh":
		return true
	default:
		return false
	}
}

func isConfigExtension(extension string) bool {
	switch extension {
	case ".cfg", ".conf", ".ini", ".json", ".properties", ".toml", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

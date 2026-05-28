package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nexusdesk/internal/buildinfo"
)

type Manifest struct {
	SchemaVersion  string `json:"schemaVersion"`
	AppID          string `json:"appId"`
	AppName        string `json:"appName"`
	Version        string `json:"version"`
	Commit         string `json:"commit"`
	BuildDate      string `json:"buildDate"`
	Platform       string `json:"platform"`
	ArtifactName   string `json:"artifactName"`
	ArtifactSize   int64  `json:"artifactSize"`
	ArtifactSHA256 string `json:"artifactSha256"`
	GeneratedAt    string `json:"generatedAt"`
}

func BuildManifest(artifactPath string, platform string, info buildinfo.Info, generatedAt time.Time) (Manifest, error) {
	artifactPath = strings.TrimSpace(artifactPath)
	if artifactPath == "" {
		return Manifest{}, errors.New("artifact path is required")
	}
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return Manifest{}, errors.New("platform is required")
	}
	if err := info.Validate(); err != nil {
		return Manifest{}, err
	}
	fileInfo, err := os.Stat(artifactPath)
	if err != nil {
		return Manifest{}, err
	}
	if fileInfo.IsDir() {
		return Manifest{}, fmt.Errorf("artifact path %q is a directory", artifactPath)
	}
	sum, err := fileSHA256(artifactPath)
	if err != nil {
		return Manifest{}, err
	}
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	return Manifest{
		SchemaVersion:  "1",
		AppID:          info.AppID,
		AppName:        info.AppName,
		Version:        strings.TrimSpace(info.Version),
		Commit:         strings.TrimSpace(info.Commit),
		BuildDate:      strings.TrimSpace(info.BuildDate),
		Platform:       platform,
		ArtifactName:   filepath.Base(artifactPath),
		ArtifactSize:   fileInfo.Size(),
		ArtifactSHA256: sum,
		GeneratedAt:    generatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func WriteManifest(path string, manifest Manifest) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("manifest output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func ReadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

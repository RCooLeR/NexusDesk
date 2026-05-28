package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nexusdesk/internal/buildinfo"
)

func TestBuildManifestHashesArtifactAndMetadata(t *testing.T) {
	root := t.TempDir()
	artifact := filepath.Join(root, "nexusdesk-test.bin")
	if err := os.WriteFile(artifact, []byte("native release artifact"), 0o644); err != nil {
		t.Fatal(err)
	}
	generatedAt := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
	manifest, err := BuildManifest(artifact, "windows", buildinfo.Info{
		AppID:     buildinfo.AppID,
		AppName:   buildinfo.AppName,
		Tagline:   buildinfo.Tagline,
		Version:   "1.2.3-beta.1",
		Commit:    "abcdef123456",
		BuildDate: "2026-05-28T11:59:00Z",
	}, generatedAt)
	if err != nil {
		t.Fatalf("BuildManifest returned error: %v", err)
	}
	if manifest.SchemaVersion != "1" || manifest.Platform != "windows" || manifest.ArtifactName != "nexusdesk-test.bin" {
		t.Fatalf("unexpected manifest identity: %#v", manifest)
	}
	if manifest.ArtifactSize != int64(len("native release artifact")) {
		t.Fatalf("unexpected artifact size: %d", manifest.ArtifactSize)
	}
	if len(manifest.ArtifactSHA256) != 64 || manifest.GeneratedAt != "2026-05-28T12:00:00Z" {
		t.Fatalf("unexpected manifest digest/time: %#v", manifest)
	}
}

func TestWriteAndReadManifestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "build", "manifest.json")
	expected := Manifest{
		SchemaVersion:  "1",
		AppID:          buildinfo.AppID,
		AppName:        buildinfo.AppName,
		Version:        "1.0.0",
		Commit:         "abc",
		BuildDate:      "2026-05-28T12:00:00Z",
		Platform:       "linux",
		ArtifactName:   "nexusdesk",
		ArtifactSize:   42,
		ArtifactSHA256: strings.Repeat("a", 64),
		GeneratedAt:    "2026-05-28T12:01:00Z",
	}
	if err := WriteManifest(path, expected); err != nil {
		t.Fatalf("WriteManifest returned error: %v", err)
	}
	got, err := ReadManifest(path)
	if err != nil {
		t.Fatalf("ReadManifest returned error: %v", err)
	}
	if got != expected {
		t.Fatalf("manifest round trip mismatch:\n got=%#v\nwant=%#v", got, expected)
	}
}

func TestBuildManifestRejectsInvalidBuildInfo(t *testing.T) {
	artifact := filepath.Join(t.TempDir(), "nexusdesk")
	if err := os.WriteFile(artifact, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := BuildManifest(artifact, "linux", buildinfo.Info{
		AppID:     buildinfo.AppID,
		AppName:   buildinfo.AppName,
		Version:   "dev",
		Commit:    "abc",
		BuildDate: "2026-05-28T12:00:00Z",
	}, time.Time{})
	if err == nil || !strings.Contains(err.Error(), "semantic version") {
		t.Fatalf("expected semantic version error, got %v", err)
	}
}

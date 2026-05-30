package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildSBOMFromGoArtifactIncludesApplicationAndModules(t *testing.T) {
	exe := currentTestExecutable(t)
	generatedAt := time.Date(2026, 5, 30, 9, 15, 0, 0, time.UTC)
	manifest, err := BuildManifest(exe, "windows", testBuildInfo(), generatedAt)
	if err != nil {
		t.Fatalf("BuildManifest returned error: %v", err)
	}
	sbom, err := BuildSBOM(exe, manifest, generatedAt)
	if err != nil {
		t.Fatalf("BuildSBOM returned error: %v", err)
	}
	if sbom.BOMFormat != "CycloneDX" || sbom.SpecVersion != "1.5" {
		t.Fatalf("unexpected SBOM header: %+v", sbom)
	}
	if sbom.Metadata.Component.Name != manifest.AppName {
		t.Fatalf("expected app component %q, got %q", manifest.AppName, sbom.Metadata.Component.Name)
	}
	if len(sbom.Metadata.Component.Hashes) == 0 || sbom.Metadata.Component.Hashes[0].Content != manifest.ArtifactSHA256 {
		t.Fatalf("expected artifact hash in app component: %+v", sbom.Metadata.Component.Hashes)
	}
	if len(sbom.Components) == 0 {
		t.Fatal("expected at least the main module component")
	}
}

func TestBuildSBOMForPackageArtifactFallsBackToManifestMetadata(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "nexusdesk-windows-installer.zip")
	if err := os.WriteFile(artifact, []byte("installer package bytes"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	generatedAt := time.Date(2026, 5, 30, 9, 20, 0, 0, time.UTC)
	manifest, err := BuildManifest(artifact, "windows-installer", testBuildInfo(), generatedAt)
	if err != nil {
		t.Fatalf("BuildManifest returned error: %v", err)
	}
	sbom, err := BuildSBOM(artifact, manifest, generatedAt)
	if err != nil {
		t.Fatalf("BuildSBOM returned error: %v", err)
	}
	if sbom.Metadata.Component.Name != manifest.AppName {
		t.Fatalf("expected app component %q, got %q", manifest.AppName, sbom.Metadata.Component.Name)
	}
	if len(sbom.Components) != 0 {
		t.Fatalf("expected package artifact fallback to omit Go module components, got %d", len(sbom.Components))
	}
	properties := []string{}
	for _, property := range sbom.Metadata.Component.Properties {
		properties = append(properties, property.Name+"="+property.Value)
	}
	if !strings.Contains(strings.Join(properties, "\n"), "nexusdesk:sbomSource=manifest-only-package-artifact") {
		t.Fatalf("expected fallback SBOM source property, got %v", properties)
	}
}

func TestWriteEvidenceSetStoresManifestSBOMAndProvenanceTogether(t *testing.T) {
	exe := currentTestExecutable(t)
	dir := t.TempDir()
	generatedAt := time.Date(2026, 5, 30, 9, 30, 0, 0, time.UTC)
	manifest, err := BuildManifest(exe, "linux", testBuildInfo(), generatedAt)
	if err != nil {
		t.Fatalf("BuildManifest returned error: %v", err)
	}
	set, err := WriteEvidenceSet(EvidenceOptions{
		ArtifactPath:     exe,
		ManifestPath:     filepath.Join(dir, "nexusdesk-linux-manifest.json"),
		Manifest:         manifest,
		GeneratedAt:      generatedAt,
		Repository:       "RCooLeR/NexusDesk",
		ReleaseWorkflow:  "ci-unix.sh",
		SourceCommitFull: "abcdef1234567890",
	})
	if err != nil {
		t.Fatalf("WriteEvidenceSet returned error: %v", err)
	}
	for _, path := range []string{set.ManifestPath, set.SBOMPath, set.ProvenancePath} {
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Fatalf("expected evidence file %s to exist, info=%v err=%v", path, info, err)
		}
	}
	if filepath.Base(set.SBOMPath) != "nexusdesk-linux-sbom.json" {
		t.Fatalf("unexpected SBOM path: %s", set.SBOMPath)
	}
	if filepath.Base(set.ProvenancePath) != "nexusdesk-linux-provenance.json" {
		t.Fatalf("unexpected provenance path: %s", set.ProvenancePath)
	}
	if set.Provenance.Subject.ArtifactSHA256 != manifest.ArtifactSHA256 {
		t.Fatalf("expected provenance subject hash %s, got %s", manifest.ArtifactSHA256, set.Provenance.Subject.ArtifactSHA256)
	}
	evidenceKinds := []string{}
	for _, evidence := range set.Provenance.Evidence {
		evidenceKinds = append(evidenceKinds, evidence.Kind)
		if strings.TrimSpace(evidence.SHA256) == "" {
			t.Fatalf("expected SHA for provenance evidence entry: %+v", evidence)
		}
	}
	joined := strings.Join(evidenceKinds, "\n")
	if !strings.Contains(joined, "release-manifest") || !strings.Contains(joined, "sbom") {
		t.Fatalf("expected manifest and SBOM evidence entries, got %v", evidenceKinds)
	}
}

func currentTestExecutable(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable returned error: %v", err)
	}
	return exe
}

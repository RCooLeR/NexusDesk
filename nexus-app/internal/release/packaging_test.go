package release

import (
	"strings"
	"testing"

	"nexusdesk/internal/buildinfo"
)

func TestEvaluatePackagingReadinessWindowsReady(t *testing.T) {
	got := EvaluatePackagingReadiness(PackagingEvidence{
		Platform:                  "windows",
		ArtifactFormat:            "msix",
		Manifest:                  testPackagingManifest("windows"),
		Signed:                    true,
		SigningIdentity:           "CN=NexusDesk Release",
		InstallerValidated:        true,
		UpdateValidated:           true,
		UninstallValidated:        true,
		CleanMachineSmokePassed:   true,
		SecretStorageSmokePassed:  true,
		AntivirusTriageDocumented: true,
	})
	if !got.Ready {
		t.Fatalf("expected ready packaging evidence, got blockers: %v", got.Blockers)
	}
	if len(got.Actions) != 0 {
		t.Fatalf("expected no required actions, got %v", got.Actions)
	}
}

func TestEvaluatePackagingReadinessBlocksUnsignedWindowsCIArtifact(t *testing.T) {
	got := EvaluatePackagingReadiness(PackagingEvidence{
		Platform:       "windows",
		ArtifactFormat: "exe",
		Manifest:       testPackagingManifest("windows"),
	})
	if got.Ready {
		t.Fatal("expected unsigned Windows CI artifact to be blocked")
	}
	expectBlocker(t, got, "Windows package is not code-signed")
	expectBlocker(t, got, "installer or package install behavior is not validated")
	expectBlocker(t, got, "update or upgrade behavior is not validated")
	expectBlocker(t, got, "uninstall and app-data retention behavior is not validated")
	expectBlocker(t, got, "protected-secret storage smoke is not verified")
	expectBlocker(t, got, "release hygiene and antivirus triage state is not documented")
}

func TestEvaluatePackagingReadinessMacRequiresNotarization(t *testing.T) {
	got := EvaluatePackagingReadiness(PackagingEvidence{
		Platform:                  "macOS",
		ArtifactFormat:            "dmg",
		Manifest:                  testPackagingManifest("darwin"),
		Signed:                    true,
		SigningIdentity:           "Developer ID Application: NexusDesk",
		InstallerValidated:        true,
		UpdateValidated:           true,
		UninstallValidated:        true,
		CleanMachineSmokePassed:   true,
		SecretStorageSmokePassed:  true,
		AntivirusTriageDocumented: true,
	})
	if got.Ready {
		t.Fatal("expected macOS package without notarization to be blocked")
	}
	expectBlocker(t, got, "macOS package is not notarized")
}

func TestEvaluatePackagingReadinessLinuxRequiresTrustStrategy(t *testing.T) {
	got := EvaluatePackagingReadiness(PackagingEvidence{
		Platform:                  "linux",
		ArtifactFormat:            "AppImage",
		Manifest:                  testPackagingManifest("linux"),
		InstallerValidated:        true,
		UpdateValidated:           true,
		UninstallValidated:        true,
		CleanMachineSmokePassed:   true,
		SecretStorageSmokePassed:  true,
		AntivirusTriageDocumented: true,
	})
	if got.Ready {
		t.Fatal("expected Linux package without trust strategy to be blocked")
	}
	expectBlocker(t, got, "Linux package trust strategy is not documented")
	if len(got.Warnings) == 0 || !strings.Contains(got.Warnings[0], "Linux package signing is not recorded") {
		t.Fatalf("expected Linux signing warning, got %v", got.Warnings)
	}
}

func TestEvaluatePackagingReadinessBlocksManifestPlatformMismatch(t *testing.T) {
	got := EvaluatePackagingReadiness(PackagingEvidence{
		Platform:                  "windows",
		ArtifactFormat:            "msix",
		Manifest:                  testPackagingManifest("linux"),
		Signed:                    true,
		SigningIdentity:           "CN=NexusDesk Release",
		InstallerValidated:        true,
		UpdateValidated:           true,
		UninstallValidated:        true,
		CleanMachineSmokePassed:   true,
		SecretStorageSmokePassed:  true,
		AntivirusTriageDocumented: true,
	})
	if got.Ready {
		t.Fatal("expected mismatched manifest platform to be blocked")
	}
	expectBlocker(t, got, `manifest platform "linux" does not match packaging platform "windows"`)
}

func TestEvaluatePackagingReadinessBlocksInvalidManifest(t *testing.T) {
	manifest := testPackagingManifest("windows")
	manifest.ArtifactSHA256 = "not-a-sha"
	got := EvaluatePackagingReadiness(PackagingEvidence{
		Platform:                  "windows",
		ArtifactFormat:            "msix",
		Manifest:                  manifest,
		Signed:                    true,
		SigningIdentity:           "CN=NexusDesk Release",
		InstallerValidated:        true,
		UpdateValidated:           true,
		UninstallValidated:        true,
		CleanMachineSmokePassed:   true,
		SecretStorageSmokePassed:  true,
		AntivirusTriageDocumented: true,
	})
	if got.Ready {
		t.Fatal("expected invalid manifest to block readiness")
	}
	expectBlocker(t, got, "release manifest artifactSha256 must be a 64-character hex digest")
}

func TestEvaluatePackagingReadinessBlocksInvalidManifestBuildMetadata(t *testing.T) {
	manifest := testPackagingManifest("windows")
	manifest.Version = "beta"
	got := EvaluatePackagingReadiness(PackagingEvidence{
		Platform:                  "windows",
		ArtifactFormat:            "msix",
		Manifest:                  manifest,
		Signed:                    true,
		SigningIdentity:           "CN=NexusDesk Release",
		InstallerValidated:        true,
		UpdateValidated:           true,
		UninstallValidated:        true,
		CleanMachineSmokePassed:   true,
		SecretStorageSmokePassed:  true,
		AntivirusTriageDocumented: true,
	})
	if got.Ready {
		t.Fatal("expected invalid build metadata to block readiness")
	}
	for _, blocker := range got.Blockers {
		if strings.Contains(blocker, "release manifest build metadata is invalid") {
			return
		}
	}
	t.Fatalf("expected build metadata blocker, got %v", got.Blockers)
}

func TestRuntimeTrustDiagnosticsRequiresReleaseEvidence(t *testing.T) {
	got := RuntimeTrustDiagnostics("windows", buildinfo.Info{
		AppID:     buildinfo.AppID,
		AppName:   buildinfo.AppName,
		Version:   "0.0.0-dev",
		Commit:    "unknown",
		BuildDate: "unknown",
	})
	if got.Ready {
		t.Fatal("expected runtime diagnostics to require attached release evidence")
	}
	if got.Platform != "windows" || got.ArtifactFormat != "runtime" {
		t.Fatalf("unexpected runtime trust target: %+v", got)
	}
	for _, blocker := range []string{
		"runtime commit metadata is not stamped",
		"runtime build date metadata is not stamped",
		"development version metadata is active",
		"packaged artifact manifest is not attached to this runtime",
		"code-signing, notarization, or package trust evidence is not attached",
		"clean-machine, update, uninstall, and protected-secret smoke evidence is not attached",
		"SBOM and provenance evidence is not attached",
	} {
		expectBlocker(t, got, blocker)
	}
}

func TestRuntimeTrustDiagnosticsRejectsUnsupportedPlatform(t *testing.T) {
	got := RuntimeTrustDiagnostics("plan9", testBuildInfo())
	if got.Ready {
		t.Fatal("expected unsupported platform to block runtime trust diagnostics")
	}
	expectBlocker(t, got, `unsupported release platform "plan9"`)
}

func testPackagingManifest(platform string) Manifest {
	return Manifest{
		SchemaVersion:  "1",
		AppID:          "com.nexusdesk.app",
		AppName:        "NexusDesk",
		Version:        "1.2.3",
		Commit:         "abcdef123456",
		BuildDate:      "2026-05-28T11:59:00Z",
		Platform:       platform,
		ArtifactName:   "nexusdesk",
		ArtifactSize:   42,
		ArtifactSHA256: strings.Repeat("a", 64),
		GeneratedAt:    "2026-05-28T12:00:00Z",
	}
}

func testBuildInfo() buildinfo.Info {
	return buildinfo.Info{
		AppID:     buildinfo.AppID,
		AppName:   buildinfo.AppName,
		Version:   "1.2.3",
		Commit:    "abcdef123456",
		BuildDate: "2026-05-28T11:59:00Z",
	}
}

func expectBlocker(t *testing.T, got PackagingReadiness, want string) {
	t.Helper()
	for _, blocker := range got.Blockers {
		if blocker == want {
			return
		}
	}
	t.Fatalf("missing blocker %q in %v", want, got.Blockers)
}

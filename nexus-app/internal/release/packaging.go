package release

import (
	"fmt"
	"strconv"
	"strings"

	"nexusdesk/internal/buildinfo"
)

type PackagingEvidence struct {
	Platform                  string
	ArtifactFormat            string
	Manifest                  Manifest
	Signed                    bool
	SigningIdentity           string
	Notarized                 bool
	PackageTrustDocumented    bool
	InstallerValidated        bool
	UpdateValidated           bool
	UninstallValidated        bool
	CleanMachineSmokePassed   bool
	SecretStorageSmokePassed  bool
	AntivirusTriageDocumented bool
}

type PackagingReadiness struct {
	Platform       string
	ArtifactFormat string
	Ready          bool
	Blockers       []string
	Warnings       []string
	Actions        []string
}

func EvaluatePackagingReadiness(e PackagingEvidence) PackagingReadiness {
	platform := normalizePlatform(e.Platform)
	format := normalizeFormat(e.ArtifactFormat)
	if platform == "" {
		platform = normalizePlatform(e.Manifest.Platform)
	}

	readiness := PackagingReadiness{
		Platform:       platform,
		ArtifactFormat: format,
	}

	if platform == "" {
		readiness.addBlocker("release platform is required", "set platform to windows, darwin, or linux")
	} else if !supportedPlatform(platform) {
		readiness.addBlocker(fmt.Sprintf("unsupported release platform %q", platform), "define packaging rules before shipping this platform")
	}

	if format == "" {
		readiness.addBlocker("artifact format is required", "record the package format produced by the release pipeline")
	} else if platform != "" && supportedPlatform(platform) && !supportedFormat(platform, format) {
		readiness.addBlocker(fmt.Sprintf("%s artifact format %q is not in the approved packaging set", platform, format), "choose an approved package format or update the platform support matrix")
	}

	for _, issue := range validateReleaseManifest(e.Manifest) {
		readiness.addBlocker(issue, "generate a valid release manifest for the packaged artifact")
	}
	if e.Manifest.Platform != "" && platform != "" && normalizePlatform(e.Manifest.Platform) != platform {
		readiness.addBlocker(fmt.Sprintf("manifest platform %q does not match packaging platform %q", e.Manifest.Platform, platform), "regenerate the manifest for the target platform")
	}

	switch platform {
	case "windows":
		if !e.Signed {
			readiness.addBlocker("Windows package is not code-signed", "sign the Windows package with the production certificate")
		}
	case "darwin":
		if !e.Signed {
			readiness.addBlocker("macOS package is not code-signed", "sign the macOS app/package with the Developer ID certificate")
		}
		if !e.Notarized {
			readiness.addBlocker("macOS package is not notarized", "notarize the macOS package and staple the ticket where applicable")
		}
	case "linux":
		if !e.PackageTrustDocumented {
			readiness.addBlocker("Linux package trust strategy is not documented", "document the chosen Linux package trust model and verification path")
		}
		if !e.Signed {
			readiness.Warnings = append(readiness.Warnings, "Linux package signing is not recorded; this is acceptable only when the documented trust strategy uses hashes or repository signing instead")
		}
	}

	if !e.InstallerValidated {
		readiness.addBlocker("installer or package install behavior is not validated", "run clean-machine install/package smoke for this platform")
	}
	if !e.UpdateValidated {
		readiness.addBlocker("update or upgrade behavior is not validated", "run update/upgrade smoke for the release channel")
	}
	if !e.UninstallValidated {
		readiness.addBlocker("uninstall and app-data retention behavior is not validated", "run uninstall/cleanup smoke and record retained data expectations")
	}
	if !e.CleanMachineSmokePassed {
		readiness.addBlocker("clean-machine smoke did not pass", "complete the clean-machine smoke checklist for this platform")
	}
	if !e.SecretStorageSmokePassed {
		readiness.addBlocker("protected-secret storage smoke is not verified", "verify Windows DPAPI, macOS Keychain, Linux Secret Service, or explicit unsupported refusal")
	}
	if !e.AntivirusTriageDocumented {
		readiness.addBlocker("release hygiene and antivirus triage state is not documented", "record antivirus/signing/trust state in release notes")
	}

	if e.Signed && strings.TrimSpace(e.SigningIdentity) == "" {
		readiness.Warnings = append(readiness.Warnings, "package is marked signed but no signing identity is recorded")
	}

	readiness.Ready = len(readiness.Blockers) == 0
	return readiness
}

func (r *PackagingReadiness) addBlocker(blocker string, action string) {
	r.Blockers = append(r.Blockers, blocker)
	r.Actions = append(r.Actions, action)
}

func validateReleaseManifest(manifest Manifest) []string {
	var issues []string
	if strings.TrimSpace(manifest.SchemaVersion) != "1" {
		issues = append(issues, "release manifest schemaVersion must be 1")
	}
	if strings.TrimSpace(manifest.AppID) == "" {
		issues = append(issues, "release manifest appId is required")
	}
	if strings.TrimSpace(manifest.AppName) == "" {
		issues = append(issues, "release manifest appName is required")
	}
	if strings.TrimSpace(manifest.Version) == "" {
		issues = append(issues, "release manifest version is required")
	}
	if strings.TrimSpace(manifest.Commit) == "" {
		issues = append(issues, "release manifest commit is required")
	}
	if strings.TrimSpace(manifest.BuildDate) == "" {
		issues = append(issues, "release manifest buildDate is required")
	}
	if normalizePlatform(manifest.Platform) == "" {
		issues = append(issues, "release manifest platform is required")
	}
	if strings.TrimSpace(manifest.ArtifactName) == "" {
		issues = append(issues, "release manifest artifactName is required")
	}
	if manifest.ArtifactSize <= 0 {
		issues = append(issues, "release manifest artifactSize must be positive")
	}
	if !validSHA256(manifest.ArtifactSHA256) {
		issues = append(issues, "release manifest artifactSha256 must be a 64-character hex digest")
	}
	if strings.TrimSpace(manifest.GeneratedAt) == "" {
		issues = append(issues, "release manifest generatedAt is required")
	}
	if err := (buildinfo.Info{
		AppID:     manifest.AppID,
		AppName:   manifest.AppName,
		Version:   manifest.Version,
		Commit:    manifest.Commit,
		BuildDate: manifest.BuildDate,
	}).Validate(); err != nil {
		issues = append(issues, "release manifest build metadata is invalid: "+err.Error())
	}
	return issues
}

func normalizePlatform(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "win", "windows":
		return "windows"
	case "mac", "macos", "darwin", "osx", "os x":
		return "darwin"
	case "linux":
		return "linux"
	default:
		return strings.ToLower(strings.TrimSpace(platform))
	}
}

func normalizeFormat(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	format = strings.TrimPrefix(format, ".")
	return format
}

func supportedPlatform(platform string) bool {
	switch platform {
	case "windows", "darwin", "linux":
		return true
	default:
		return false
	}
}

func supportedFormat(platform string, format string) bool {
	approved := map[string][]string{
		"windows": {"exe", "msi", "msix", "zip"},
		"darwin":  {"app", "dmg", "pkg", "zip"},
		"linux":   {"appimage", "deb", "rpm", "tar.gz", "zip"},
	}
	for _, allowed := range approved[platform] {
		if format == allowed {
			return true
		}
	}
	return false
}

func validSHA256(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 64 {
		return false
	}
	_, err := strconv.ParseUint(value[:16], 16, 64)
	if err != nil {
		return false
	}
	_, err = strconv.ParseUint(value[16:32], 16, 64)
	if err != nil {
		return false
	}
	_, err = strconv.ParseUint(value[32:48], 16, 64)
	if err != nil {
		return false
	}
	_, err = strconv.ParseUint(value[48:], 16, 64)
	return err == nil
}

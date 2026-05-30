package release

import (
	"crypto/sha256"
	debugBuildInfo "debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	runtimeDebug "runtime/debug"
	"strings"
	"time"
)

type EvidenceOptions struct {
	ArtifactPath     string
	ManifestPath     string
	SBOMPath         string
	ProvenancePath   string
	Manifest         Manifest
	GeneratedAt      time.Time
	Generator        string
	Repository       string
	ReleaseWorkflow  string
	SourceCommitFull string
}

type EvidenceSet struct {
	ManifestPath   string
	SBOMPath       string
	ProvenancePath string
	Manifest       Manifest
	SBOM           SBOM
	Provenance     Provenance
}

type SBOM struct {
	BOMFormat    string          `json:"bomFormat"`
	SpecVersion  string          `json:"specVersion"`
	SerialNumber string          `json:"serialNumber"`
	Version      int             `json:"version"`
	Metadata     SBOMMetadata    `json:"metadata"`
	Components   []SBOMComponent `json:"components"`
}

type SBOMMetadata struct {
	Timestamp  string         `json:"timestamp"`
	Tools      []SBOMTool     `json:"tools"`
	Component  SBOMComponent  `json:"component"`
	Properties []SBOMProperty `json:"properties,omitempty"`
}

type SBOMTool struct {
	Vendor  string `json:"vendor"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type SBOMComponent struct {
	Type       string         `json:"type"`
	Name       string         `json:"name"`
	Version    string         `json:"version,omitempty"`
	Scope      string         `json:"scope,omitempty"`
	PURL       string         `json:"purl,omitempty"`
	Hashes     []SBOMHash     `json:"hashes,omitempty"`
	Properties []SBOMProperty `json:"properties,omitempty"`
}

type SBOMHash struct {
	Alg     string `json:"alg"`
	Content string `json:"content"`
}

type SBOMProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Provenance struct {
	SchemaVersion string               `json:"schemaVersion"`
	Subject       ProvenanceSubject    `json:"subject"`
	Build         ProvenanceBuild      `json:"build"`
	Evidence      []ProvenanceEvidence `json:"evidence"`
	GeneratedAt   string               `json:"generatedAt"`
	Generator     string               `json:"generator"`
}

type ProvenanceSubject struct {
	AppID          string `json:"appId"`
	AppName        string `json:"appName"`
	Version        string `json:"version"`
	Commit         string `json:"commit"`
	BuildDate      string `json:"buildDate"`
	Platform       string `json:"platform"`
	ArtifactName   string `json:"artifactName"`
	ArtifactSize   int64  `json:"artifactSize"`
	ArtifactSHA256 string `json:"artifactSha256"`
}

type ProvenanceBuild struct {
	Repository       string `json:"repository,omitempty"`
	Workflow         string `json:"workflow,omitempty"`
	SourceCommitFull string `json:"sourceCommitFull,omitempty"`
}

type ProvenanceEvidence struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

func WriteEvidenceSet(options EvidenceOptions) (EvidenceSet, error) {
	options.ManifestPath = strings.TrimSpace(options.ManifestPath)
	if options.ManifestPath == "" {
		return EvidenceSet{}, errors.New("manifest path is required")
	}
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	if strings.TrimSpace(options.Generator) == "" {
		options.Generator = "nexusdesk-release-evidence"
	}
	options.SBOMPath = firstEvidencePath(options.SBOMPath, derivedEvidencePath(options.ManifestPath, "sbom"))
	options.ProvenancePath = firstEvidencePath(options.ProvenancePath, derivedEvidencePath(options.ManifestPath, "provenance"))

	if err := WriteManifest(options.ManifestPath, options.Manifest); err != nil {
		return EvidenceSet{}, err
	}
	sbom, err := BuildSBOM(options.ArtifactPath, options.Manifest, options.GeneratedAt)
	if err != nil {
		return EvidenceSet{}, err
	}
	if err := WriteSBOM(options.SBOMPath, sbom); err != nil {
		return EvidenceSet{}, err
	}
	provenance, err := BuildProvenance(options)
	if err != nil {
		return EvidenceSet{}, err
	}
	if err := WriteProvenance(options.ProvenancePath, provenance); err != nil {
		return EvidenceSet{}, err
	}
	return EvidenceSet{
		ManifestPath:   options.ManifestPath,
		SBOMPath:       options.SBOMPath,
		ProvenancePath: options.ProvenancePath,
		Manifest:       options.Manifest,
		SBOM:           sbom,
		Provenance:     provenance,
	}, nil
}

func BuildSBOM(artifactPath string, manifest Manifest, generatedAt time.Time) (SBOM, error) {
	artifactPath = strings.TrimSpace(artifactPath)
	if artifactPath == "" {
		return SBOM{}, errors.New("artifact path is required")
	}
	info, err := debugBuildInfo.ReadFile(artifactPath)
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	componentName := strings.TrimSpace(manifest.AppName)
	if err == nil {
		componentName = firstNonEmpty(componentName, info.Main.Path)
	}
	appComponent := SBOMComponent{
		Type:    "application",
		Name:    firstNonEmpty(componentName, "nexusdesk"),
		Version: strings.TrimSpace(manifest.Version),
		Hashes:  []SBOMHash{{Alg: "SHA-256", Content: strings.TrimSpace(manifest.ArtifactSHA256)}},
		Properties: []SBOMProperty{
			{Name: "nexusdesk:appId", Value: strings.TrimSpace(manifest.AppID)},
			{Name: "nexusdesk:platform", Value: strings.TrimSpace(manifest.Platform)},
			{Name: "nexusdesk:commit", Value: strings.TrimSpace(manifest.Commit)},
		},
	}
	components := []SBOMComponent{}
	if err == nil {
		components = make([]SBOMComponent, 0, len(info.Deps)+1)
		components = append(components, moduleComponent(info.Main, "required"))
		for _, dep := range info.Deps {
			components = append(components, moduleComponent(*dep, "required"))
		}
	} else {
		appComponent.Properties = append(appComponent.Properties,
			SBOMProperty{Name: "nexusdesk:sbomSource", Value: "manifest-only-package-artifact"},
			SBOMProperty{Name: "nexusdesk:goBuildInfo", Value: "unavailable: " + err.Error()},
		)
	}
	return SBOM{
		BOMFormat:    "CycloneDX",
		SpecVersion:  "1.5",
		SerialNumber: "urn:uuid:" + deterministicUUID(manifest.ArtifactSHA256+generatedAt.UTC().Format(time.RFC3339)),
		Version:      1,
		Metadata: SBOMMetadata{
			Timestamp: generatedAt.UTC().Format(time.RFC3339),
			Tools: []SBOMTool{{
				Vendor:  "NexusDesk",
				Name:    "release-manifest",
				Version: "1",
			}},
			Component: appComponent,
			Properties: []SBOMProperty{
				{Name: "nexusdesk:evidence", Value: "release-sbom"},
			},
		},
		Components: components,
	}, nil
}

func WriteSBOM(path string, sbom SBOM) error {
	return writeJSONFile(path, sbom)
}

func BuildProvenance(options EvidenceOptions) (Provenance, error) {
	if strings.TrimSpace(options.ManifestPath) == "" || strings.TrimSpace(options.SBOMPath) == "" {
		return Provenance{}, errors.New("manifest and SBOM paths are required")
	}
	manifestSHA, err := fileSHA256(options.ManifestPath)
	if err != nil {
		return Provenance{}, err
	}
	sbomSHA, err := fileSHA256(options.SBOMPath)
	if err != nil {
		return Provenance{}, err
	}
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	if strings.TrimSpace(options.Generator) == "" {
		options.Generator = "nexusdesk-release-evidence"
	}
	manifest := options.Manifest
	return Provenance{
		SchemaVersion: "1",
		Subject: ProvenanceSubject{
			AppID:          manifest.AppID,
			AppName:        manifest.AppName,
			Version:        manifest.Version,
			Commit:         manifest.Commit,
			BuildDate:      manifest.BuildDate,
			Platform:       manifest.Platform,
			ArtifactName:   manifest.ArtifactName,
			ArtifactSize:   manifest.ArtifactSize,
			ArtifactSHA256: manifest.ArtifactSHA256,
		},
		Build: ProvenanceBuild{
			Repository:       strings.TrimSpace(options.Repository),
			Workflow:         strings.TrimSpace(options.ReleaseWorkflow),
			SourceCommitFull: strings.TrimSpace(options.SourceCommitFull),
		},
		Evidence: []ProvenanceEvidence{
			{Kind: "release-manifest", Path: filepath.Base(options.ManifestPath), SHA256: manifestSHA},
			{Kind: "sbom", Path: filepath.Base(options.SBOMPath), SHA256: sbomSHA},
		},
		GeneratedAt: options.GeneratedAt.UTC().Format(time.RFC3339),
		Generator:   options.Generator,
	}, nil
}

func WriteProvenance(path string, provenance Provenance) error {
	return writeJSONFile(path, provenance)
}

func writeJSONFile(path string, value any) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("JSON output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func moduleComponent(module runtimeDebug.Module, scope string) SBOMComponent {
	version := strings.TrimSpace(module.Version)
	properties := []SBOMProperty{}
	if module.Replace != nil {
		properties = append(properties, SBOMProperty{Name: "go:replace", Value: strings.TrimSpace(module.Replace.Path + " " + module.Replace.Version)})
	}
	return SBOMComponent{
		Type:       "library",
		Name:       strings.TrimSpace(module.Path),
		Version:    version,
		Scope:      scope,
		PURL:       golangPURL(module.Path, version),
		Properties: properties,
	}
}

func golangPURL(path string, version string) string {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return ""
	}
	if strings.TrimSpace(version) == "" {
		return "pkg:golang/" + path
	}
	return "pkg:golang/" + path + "@" + strings.TrimSpace(version)
}

func derivedEvidencePath(manifestPath string, kind string) string {
	dir := filepath.Dir(manifestPath)
	base := strings.TrimSuffix(filepath.Base(manifestPath), ".json")
	base = strings.TrimSuffix(base, "-manifest")
	if strings.TrimSpace(base) == "" || base == "." {
		base = "release"
	}
	return filepath.Join(dir, base+"-"+kind+".json")
}

func firstEvidencePath(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func deterministicUUID(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	bytes := sum[:16]
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	hexValue := hex.EncodeToString(bytes)
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexValue[0:8], hexValue[8:12], hexValue[12:16], hexValue[16:20], hexValue[20:32])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

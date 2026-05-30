package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"nexusdesk/internal/buildinfo"
	"nexusdesk/internal/release"
)

func main() {
	artifact := flag.String("artifact", "", "Path to the built release artifact")
	output := flag.String("output", "", "Path for the JSON release manifest")
	sbomOutput := flag.String("sbom-output", "", "Path for the JSON release SBOM; defaults next to the manifest")
	provenanceOutput := flag.String("provenance-output", "", "Path for the JSON release provenance; defaults next to the manifest")
	platform := flag.String("platform", "", "Release platform identifier")
	version := flag.String("version", buildinfo.Current().Version, "Release semantic version")
	commit := flag.String("commit", buildinfo.Current().Commit, "Release commit")
	buildDate := flag.String("build-date", buildinfo.Current().BuildDate, "Release build date in RFC3339 format")
	repository := flag.String("repository", "", "Repository or source identifier for provenance")
	workflow := flag.String("workflow", "", "Release workflow identifier for provenance")
	sourceCommitFull := flag.String("source-commit-full", "", "Full source commit for provenance")
	flag.Parse()

	info := buildinfo.Current()
	info.Version = strings.TrimSpace(*version)
	info.Commit = strings.TrimSpace(*commit)
	info.BuildDate = strings.TrimSpace(*buildDate)
	generatedAt := time.Now().UTC()
	manifest, err := release.BuildManifest(*artifact, *platform, info, generatedAt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	evidence, err := release.WriteEvidenceSet(release.EvidenceOptions{
		ArtifactPath:     *artifact,
		ManifestPath:     *output,
		SBOMPath:         *sbomOutput,
		ProvenancePath:   *provenanceOutput,
		Manifest:         manifest,
		GeneratedAt:      generatedAt,
		Generator:        "cmd/release-manifest",
		Repository:       *repository,
		ReleaseWorkflow:  *workflow,
		SourceCommitFull: *sourceCommitFull,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Wrote release manifest: %s\n", evidence.ManifestPath)
	fmt.Printf("Wrote release SBOM: %s\n", evidence.SBOMPath)
	fmt.Printf("Wrote release provenance: %s\n", evidence.ProvenancePath)
	fmt.Printf("Artifact SHA256: %s\n", manifest.ArtifactSHA256)
}

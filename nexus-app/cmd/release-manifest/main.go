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
	platform := flag.String("platform", "", "Release platform identifier")
	version := flag.String("version", buildinfo.Current().Version, "Release semantic version")
	commit := flag.String("commit", buildinfo.Current().Commit, "Release commit")
	buildDate := flag.String("build-date", buildinfo.Current().BuildDate, "Release build date in RFC3339 format")
	flag.Parse()

	info := buildinfo.Current()
	info.Version = strings.TrimSpace(*version)
	info.Commit = strings.TrimSpace(*commit)
	info.BuildDate = strings.TrimSpace(*buildDate)
	manifest, err := release.BuildManifest(*artifact, *platform, info, time.Now().UTC())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := release.WriteManifest(*output, manifest); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Wrote release manifest: %s\n", *output)
	fmt.Printf("Artifact SHA256: %s\n", manifest.ArtifactSHA256)
}

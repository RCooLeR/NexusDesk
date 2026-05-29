package artifacts

import (
	"fmt"
	"strings"
	"time"
)

const (
	ProvenanceStatusOK      = "ok"
	ProvenanceStatusWarning = "warning"
)

type ProvenanceIssue struct {
	RelPath string
	Kind    string
	Message string
}

type ProvenanceSummary struct {
	CheckedAt       time.Time
	ArtifactCount   int
	WithMetadata    int
	WithLineage     int
	MissingMetadata int
	MissingLineage  int
	Issues          []ProvenanceIssue
}

func (summary ProvenanceSummary) Status() string {
	if len(summary.Issues) > 0 {
		return ProvenanceStatusWarning
	}
	return ProvenanceStatusOK
}

func (summary ProvenanceSummary) Message() string {
	if summary.ArtifactCount == 0 {
		return "No native artifacts are present yet."
	}
	if summary.Status() == ProvenanceStatusOK {
		return fmt.Sprintf("%d artifact(s) have readable metadata and provenance signals.", summary.ArtifactCount)
	}
	return fmt.Sprintf("%d artifact(s) checked, %d provenance issue(s) found.", summary.ArtifactCount, len(summary.Issues))
}

func (s *Store) InspectProvenance(options ListOptions) (ProvenanceSummary, error) {
	artifacts, err := s.ListArtifacts(options)
	if err != nil {
		return ProvenanceSummary{}, err
	}
	summary := ProvenanceSummary{
		CheckedAt:     time.Now().UTC(),
		ArtifactCount: len(artifacts),
	}
	for _, artifact := range artifacts {
		metadata, err := s.ReadArtifactMetadata(artifact.RelPath)
		if err != nil {
			summary.MissingMetadata++
			summary.Issues = append(summary.Issues, ProvenanceIssue{
				RelPath: artifact.RelPath,
				Kind:    firstNonEmptyArtifact(artifact.Kind, inferKind(artifact.RelPath)),
				Message: "metadata sidecar is missing or unreadable",
			})
			continue
		}
		summary.WithMetadata++
		if issue := validateArtifactMetadataProvenance(artifact.RelPath, metadata); issue.Message != "" {
			if issue.RelPath == "" {
				issue.RelPath = artifact.RelPath
			}
			if issue.Kind == "" {
				issue.Kind = firstNonEmptyArtifact(metadata.Kind, artifact.Kind)
			}
			summary.MissingLineage++
			summary.Issues = append(summary.Issues, issue)
			continue
		}
		summary.WithLineage++
	}
	return summary, nil
}

func validateArtifactMetadataProvenance(relPath string, metadata Metadata) ProvenanceIssue {
	kind := strings.TrimSpace(metadata.Kind)
	switch {
	case kind == "":
		return ProvenanceIssue{RelPath: relPath, Message: "metadata is missing artifact kind"}
	case strings.TrimSpace(metadata.Title) == "":
		return ProvenanceIssue{RelPath: relPath, Kind: kind, Message: "metadata is missing artifact title"}
	case strings.TrimSpace(metadata.RelPath) == "":
		return ProvenanceIssue{RelPath: relPath, Kind: kind, Message: "metadata is missing artifact relPath"}
	case metadata.GeneratedAt.IsZero():
		return ProvenanceIssue{RelPath: relPath, Kind: kind, Message: "metadata is missing generatedAt timestamp"}
	case !artifactMetadataHasLineage(metadata):
		return ProvenanceIssue{RelPath: relPath, Kind: kind, Message: "metadata is missing source, job, prompt, query, package, or tool-run lineage"}
	default:
		return ProvenanceIssue{}
	}
}

func artifactMetadataHasLineage(metadata Metadata) bool {
	if strings.TrimSpace(metadata.Source) != "" ||
		strings.TrimSpace(metadata.ContextRelPath) != "" ||
		strings.TrimSpace(metadata.Prompt) != "" ||
		strings.TrimSpace(metadata.Model) != "" ||
		strings.TrimSpace(metadata.ModelRouteID) != "" ||
		strings.TrimSpace(metadata.ModelRoute) != "" ||
		strings.TrimSpace(metadata.JobID) != "" ||
		strings.TrimSpace(metadata.TaskID) != "" ||
		strings.TrimSpace(metadata.ExportFormat) != "" ||
		strings.TrimSpace(metadata.ExportTemplate) != "" ||
		strings.TrimSpace(metadata.ThemeName) != "" {
		return true
	}
	return len(metadata.SourcePaths) > 0 ||
		len(metadata.CitationRefs) > 0 ||
		len(metadata.CitedSourcePaths) > 0 ||
		len(metadata.UncitedSourcePaths) > 0 ||
		len(metadata.SourceFingerprints) > 0 ||
		len(metadata.PackageFiles) > 0 ||
		metadata.PackageValidation != nil
}

func FormatProvenanceSummary(summary ProvenanceSummary, limit int) string {
	var builder strings.Builder
	builder.WriteString("Artifact provenance: ")
	builder.WriteString(summary.Message())
	builder.WriteString("\n")
	builder.WriteString(fmt.Sprintf("Artifacts: %d | metadata: %d | lineage: %d | missing metadata: %d | missing lineage: %d\n", summary.ArtifactCount, summary.WithMetadata, summary.WithLineage, summary.MissingMetadata, summary.MissingLineage))
	if len(summary.Issues) == 0 {
		return builder.String()
	}
	if limit <= 0 || limit > len(summary.Issues) {
		limit = len(summary.Issues)
	}
	builder.WriteString("Issues:\n")
	for index := 0; index < limit; index++ {
		issue := summary.Issues[index]
		builder.WriteString("- ")
		builder.WriteString(firstNonEmptyArtifact(issue.RelPath, "(unknown artifact)"))
		if strings.TrimSpace(issue.Kind) != "" {
			builder.WriteString(" [")
			builder.WriteString(issue.Kind)
			builder.WriteString("]")
		}
		builder.WriteString(": ")
		builder.WriteString(issue.Message)
		builder.WriteString("\n")
	}
	if len(summary.Issues) > limit {
		builder.WriteString(fmt.Sprintf("... %d more issue(s)\n", len(summary.Issues)-limit))
	}
	return builder.String()
}

package artifacts

import (
	"path/filepath"
	"strings"
)

func (s *Store) Lineage(relPath string) (Lineage, error) {
	artifact, err := s.artifactByPath(relPath)
	if err != nil {
		return Lineage{}, err
	}
	return lineageForArtifact(artifact), nil
}

func lineageForArtifact(artifact Artifact) Lineage {
	artifactID := "artifact:" + artifact.RelPath
	lineage := Lineage{
		Nodes: []LineageNode{{ID: artifactID, Kind: "artifact", Label: artifactTitle(artifact), RelPath: artifact.RelPath}},
	}
	if artifact.JobID != "" {
		lineage.Nodes = append(lineage.Nodes, LineageNode{ID: "job:" + artifact.JobID, Kind: "job", Label: artifact.JobID})
		lineage.Edges = append(lineage.Edges, LineageEdge{From: "job:" + artifact.JobID, To: artifactID, Label: "generated"})
	}
	if artifact.TaskID != "" {
		lineage.Nodes = append(lineage.Nodes, LineageNode{ID: "task:" + artifact.TaskID, Kind: "task", Label: artifact.TaskID})
		target := artifactID
		if artifact.JobID != "" {
			target = "job:" + artifact.JobID
		}
		lineage.Edges = append(lineage.Edges, LineageEdge{From: "task:" + artifact.TaskID, To: target, Label: "ran"})
	}
	for _, source := range lineageSources(artifact) {
		id := "source:" + source
		lineage.Nodes = append(lineage.Nodes, LineageNode{ID: id, Kind: "source", Label: filepath.Base(source), RelPath: source})
		lineage.Edges = append(lineage.Edges, LineageEdge{From: id, To: artifactID, Label: "cited"})
	}
	return summarizeLineage(lineage)
}

func lineageSources(artifact Artifact) []string {
	sources := []string{}
	seen := map[string]bool{}
	for _, source := range artifact.SourcePaths {
		source = filepath.ToSlash(strings.TrimSpace(source))
		if source != "" && !seen[source] {
			seen[source] = true
			sources = append(sources, source)
		}
	}
	if len(sources) == 0 && strings.TrimSpace(artifact.Source) != "" {
		source := filepath.ToSlash(strings.TrimSpace(artifact.Source))
		seen[source] = true
		sources = append(sources, source)
	}
	return sources
}

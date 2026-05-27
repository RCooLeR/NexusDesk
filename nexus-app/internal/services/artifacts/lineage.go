package artifacts

import "strings"

func (s *Store) Lineage(relPath string) (Lineage, error) {
	artifact, err := s.artifactByPath(relPath)
	if err != nil {
		return Lineage{}, err
	}
	lineage := Lineage{
		Nodes: []LineageNode{{ID: artifact.RelPath, Kind: "artifact", Label: artifactTitle(artifact)}},
	}
	if artifact.JobID != "" {
		lineage.Nodes = append(lineage.Nodes, LineageNode{ID: "job:" + artifact.JobID, Kind: "job", Label: artifact.JobID})
		lineage.Edges = append(lineage.Edges, LineageEdge{From: "job:" + artifact.JobID, To: artifact.RelPath, Label: "generated"})
	}
	if artifact.TaskID != "" {
		lineage.Nodes = append(lineage.Nodes, LineageNode{ID: "task:" + artifact.TaskID, Kind: "task", Label: artifact.TaskID})
		target := artifact.RelPath
		if artifact.JobID != "" {
			target = "job:" + artifact.JobID
		}
		lineage.Edges = append(lineage.Edges, LineageEdge{From: "task:" + artifact.TaskID, To: target, Label: "ran"})
	}
	for _, source := range lineageSources(artifact) {
		id := "source:" + source
		lineage.Nodes = append(lineage.Nodes, LineageNode{ID: id, Kind: "source", Label: source})
		lineage.Edges = append(lineage.Edges, LineageEdge{From: id, To: artifact.RelPath, Label: "cited"})
	}
	return lineage, nil
}

func lineageSources(artifact Artifact) []string {
	sources := []string{}
	if strings.TrimSpace(artifact.Source) != "" {
		sources = append(sources, strings.TrimSpace(artifact.Source))
	}
	for _, source := range artifact.SourcePaths {
		source = strings.TrimSpace(source)
		if source != "" {
			sources = append(sources, source)
		}
	}
	return sources
}

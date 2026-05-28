package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (s *Store) LineageGraph(options ListOptions) (Lineage, error) {
	artifacts, err := s.ListArtifacts(options)
	if err != nil {
		return Lineage{}, err
	}
	graph := Lineage{}
	nodes := map[string]LineageNode{}
	edges := map[string]LineageEdge{}
	for _, artifact := range artifacts {
		lineage := lineageForArtifact(artifact)
		for _, node := range lineage.Nodes {
			addLineageNode(nodes, node)
		}
		for _, edge := range lineage.Edges {
			addLineageEdge(edges, edge)
		}
	}
	for _, node := range nodes {
		graph.Nodes = append(graph.Nodes, node)
	}
	for _, edge := range edges {
		graph.Edges = append(graph.Edges, edge)
	}
	sortLineage(graph.Nodes, graph.Edges)
	return summarizeLineage(graph), nil
}

func (s *Store) WriteLineageGraphArtifact(lineage Lineage) (Artifact, error) {
	lineage = summarizeLineage(lineage)
	createdAt := time.Now().UTC()
	relPath := s.relPath("lineage", fmt.Sprintf("%s-artifact-lineage.json", artifactTimestamp(createdAt)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	payload, err := json.MarshalIndent(lineage, "", "  ")
	if err != nil {
		return Artifact{}, err
	}
	payload = append(payload, '\n')
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	if _, err := file.Write(payload); err != nil {
		return Artifact{}, err
	}
	sourcePaths := lineageSourcePaths(lineage)
	metadata := Metadata{
		Kind:        "artifact-lineage",
		Title:       "Artifact Lineage Graph",
		RelPath:     relPath,
		Source:      "artifact lineage",
		SourcePaths: sourcePaths,
		GeneratedAt: createdAt,
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:         metadata.Kind,
		Title:        metadata.Title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      "Artifact lineage graph exported to " + relPath + ".",
		Size:         int64(len(payload)),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  sourcePaths,
	}, nil
}

func ParseLineageJSON(content string, relPath string) (LineageImport, error) {
	var lineage Lineage
	if err := json.Unmarshal([]byte(content), &lineage); err != nil {
		return LineageImport{}, err
	}
	lineage = summarizeLineage(lineage)
	source := filepath.ToSlash(strings.TrimSpace(relPath))
	if source == "" {
		source = "selected JSON"
	}
	return LineageImport{
		Lineage: lineage,
		Message: fmt.Sprintf("Imported %d lineage nodes and %d relationships from %s.", len(lineage.Nodes), len(lineage.Edges), source),
	}, nil
}

func summarizeLineage(lineage Lineage) Lineage {
	sortLineage(lineage.Nodes, lineage.Edges)
	counts := map[string]int{}
	for _, edge := range lineage.Edges {
		label := strings.TrimSpace(edge.Label)
		if label == "" {
			label = "related"
		}
		counts[label]++
	}
	lineage.RelationshipCounts = counts
	lineage.Message = fmt.Sprintf("%d lineage nodes and %d relationships.", len(lineage.Nodes), len(lineage.Edges))
	return lineage
}

func addLineageNode(nodes map[string]LineageNode, node LineageNode) {
	node.ID = strings.TrimSpace(node.ID)
	if node.ID == "" {
		return
	}
	if _, exists := nodes[node.ID]; !exists {
		nodes[node.ID] = node
	}
}

func addLineageEdge(edges map[string]LineageEdge, edge LineageEdge) {
	edge.From = strings.TrimSpace(edge.From)
	edge.To = strings.TrimSpace(edge.To)
	if edge.From == "" || edge.To == "" {
		return
	}
	key := edge.From + "\x00" + edge.Label + "\x00" + edge.To
	if _, exists := edges[key]; !exists {
		edges[key] = edge
	}
}

func sortLineage(nodes []LineageNode, edges []LineageEdge) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind < nodes[j].Kind
		}
		if nodes[i].Label != nodes[j].Label {
			return nodes[i].Label < nodes[j].Label
		}
		return nodes[i].ID < nodes[j].ID
	})
	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].Label != edges[j].Label {
			return edges[i].Label < edges[j].Label
		}
		return edges[i].To < edges[j].To
	})
}

func lineageSourcePaths(lineage Lineage) []string {
	paths := []string{}
	seen := map[string]bool{}
	for _, node := range lineage.Nodes {
		relPath := filepath.ToSlash(strings.TrimSpace(node.RelPath))
		if relPath == "" || seen[relPath] {
			continue
		}
		seen[relPath] = true
		paths = append(paths, relPath)
	}
	sort.Strings(paths)
	return paths
}

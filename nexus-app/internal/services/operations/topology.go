package operations

import (
	"fmt"
	"sort"
	"strings"
)

func BuildComposeTopology(services []ComposeService) ComposeTopology {
	topology := ComposeTopology{
		Services:     make([]ComposeTopologyService, 0, len(services)),
		Edges:        []ComposeTopologyEdge{},
		ExposedPorts: []ComposePortExposure{},
		NamedVolumes: []string{},
		Warnings:     []string{},
	}
	if len(services) == 0 {
		topology.Summary = "No Compose services were detected."
		topology.Warnings = append(topology.Warnings, "No services were found in the inspected Compose file.")
		return topology
	}

	knownServices := map[string]struct{}{}
	namedVolumes := map[string]struct{}{}
	for _, service := range services {
		name := strings.TrimSpace(service.Name)
		if name == "" {
			continue
		}
		knownServices[name] = struct{}{}
		topology.Services = append(topology.Services, ComposeTopologyService{
			Name:    name,
			Image:   strings.TrimSpace(service.Image),
			Ports:   cleanList(service.Ports),
			Volumes: cleanList(service.Volumes),
		})
		for _, port := range cleanList(service.Ports) {
			topology.ExposedPorts = append(topology.ExposedPorts, ComposePortExposure{Service: name, Port: port})
		}
		for _, volume := range cleanList(service.Volumes) {
			if named := namedVolumeSource(volume); named != "" {
				namedVolumes[named] = struct{}{}
			}
		}
	}

	for _, service := range services {
		from := strings.TrimSpace(service.Name)
		if from == "" {
			continue
		}
		for _, dependency := range cleanList(service.DependsOn) {
			_, ok := knownServices[dependency]
			topology.Edges = append(topology.Edges, ComposeTopologyEdge{
				From:     from,
				To:       dependency,
				Relation: "depends_on",
				Missing:  !ok,
			})
			if !ok {
				topology.Warnings = append(topology.Warnings, fmt.Sprintf("Service %q depends on %q, but that service was not found.", from, dependency))
			}
		}
	}

	sort.Slice(topology.Services, func(left, right int) bool {
		return strings.ToLower(topology.Services[left].Name) < strings.ToLower(topology.Services[right].Name)
	})
	sort.Slice(topology.Edges, func(left, right int) bool {
		leftKey := topology.Edges[left].From + "\x00" + topology.Edges[left].To
		rightKey := topology.Edges[right].From + "\x00" + topology.Edges[right].To
		return strings.ToLower(leftKey) < strings.ToLower(rightKey)
	})
	sort.Slice(topology.ExposedPorts, func(left, right int) bool {
		leftKey := topology.ExposedPorts[left].Service + "\x00" + topology.ExposedPorts[left].Port
		rightKey := topology.ExposedPorts[right].Service + "\x00" + topology.ExposedPorts[right].Port
		return strings.ToLower(leftKey) < strings.ToLower(rightKey)
	})
	topology.NamedVolumes = sortedKeys(namedVolumes)
	topology.Warnings = sortedUnique(topology.Warnings)
	topology.Summary = fmt.Sprintf("%d service(s), %d dependency edge(s), %d exposed port(s), %d named volume(s).",
		len(topology.Services),
		len(topology.Edges),
		len(topology.ExposedPorts),
		len(topology.NamedVolumes),
	)
	return topology
}

func cleanList(values []string) []string {
	seen := map[string]struct{}{}
	cleaned := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func namedVolumeSource(volume string) string {
	volume = strings.TrimSpace(volume)
	if volume == "" {
		return ""
	}
	source := volume
	if index := strings.Index(volume, ":"); index >= 0 {
		source = strings.TrimSpace(volume[:index])
	}
	if source == "" || strings.HasPrefix(source, ".") || strings.HasPrefix(source, "/") || strings.HasPrefix(source, "~") || strings.HasPrefix(source, "$") {
		return ""
	}
	if len(source) >= 2 && source[1] == ':' {
		return ""
	}
	return source
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedUnique(values []string) []string {
	seen := map[string]struct{}{}
	unique := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	sort.Strings(unique)
	return unique
}

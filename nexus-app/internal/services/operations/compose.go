package operations

import "strings"

func ParseComposeServices(content string) []ComposeService {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	services := []ComposeService{}
	inServices := false
	var current *ComposeService
	var currentList string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 0 && trimmed == "services:" {
			inServices = true
			current = nil
			currentList = ""
			continue
		}
		if !inServices {
			continue
		}
		if indent == 0 && !strings.HasPrefix(trimmed, "services:") {
			break
		}
		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			name := strings.TrimSuffix(trimmed, ":")
			services = append(services, ComposeService{Name: name})
			current = &services[len(services)-1]
			currentList = ""
			continue
		}
		if current == nil || indent < 4 {
			continue
		}
		if currentList == "depends_on" && indent >= 6 && strings.HasSuffix(trimmed, ":") {
			dependency := strings.TrimSuffix(trimmed, ":")
			if dependency != "" {
				current.DependsOn = append(current.DependsOn, dependency)
			}
			continue
		}
		if key, value, ok := splitComposeKey(trimmed); ok {
			currentList = ""
			switch key {
			case "image":
				current.Image = stripComposeValue(value)
			case "ports":
				currentList = "ports"
				appendInlineComposeList(&current.Ports, value)
			case "volumes":
				currentList = "volumes"
				appendInlineComposeList(&current.Volumes, value)
			case "depends_on":
				currentList = "depends_on"
				appendInlineComposeList(&current.DependsOn, value)
			}
			continue
		}
		if currentList != "" && strings.HasPrefix(trimmed, "- ") {
			appendComposeListItem(current, currentList, strings.TrimPrefix(trimmed, "- "))
		}
	}
	return services
}

func splitComposeKey(line string) (string, string, bool) {
	index := strings.Index(line, ":")
	if index < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:index])
	value := strings.TrimSpace(line[index+1:])
	return key, value, key != ""
}

func appendInlineComposeList(target *[]string, value string) {
	value = stripComposeValue(value)
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		return
	}
	for _, item := range strings.Split(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"), ",") {
		cleaned := stripComposeValue(item)
		if cleaned != "" {
			*target = append(*target, cleaned)
		}
	}
}

func appendComposeListItem(service *ComposeService, list string, value string) {
	value = stripComposeValue(value)
	if value == "" {
		return
	}
	switch list {
	case "ports":
		service.Ports = append(service.Ports, value)
	case "volumes":
		service.Volumes = append(service.Volumes, value)
	case "depends_on":
		service.DependsOn = append(service.DependsOn, value)
	}
}

func stripComposeValue(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"'`)
}

package llm

import (
	"fmt"
	"strings"
)

// ProviderGuidance converts probe failures into concrete remediation steps that
// can be reused by Settings, Diagnostics, and future background health checks.
func ProviderGuidance(config Config, result ProbeResult, probeErr error) []string {
	config = normalizeConfig(config)
	actions := guidanceAccumulator{}
	text := strings.ToLower(strings.Join([]string{
		config.Provider,
		config.Protocol,
		config.BaseURL,
		result.Message,
		result.Endpoint,
		errorString(probeErr),
		strings.Join(result.Warnings, " "),
	}, " "))
	ollama := isOllamaConfig(config, result)

	if probeErr != nil {
		actions.add("Check provider base URL and network reachability: " + config.BaseURL + ".")
		if ollama {
			actions.add(`For Ollama, start the runtime with "ollama serve" and verify installed models with "ollama list".`)
		}
		if containsAny(text, "connection refused", "no such host", "timeout", "deadline exceeded") {
			actions.add("Confirm the local runtime or remote endpoint is running before retrying the provider probe.")
		}
	}

	if !result.OK && strings.TrimSpace(result.Message) != "" {
		switch {
		case containsAny(text, "http 401", "http 403", "unauthorized", "forbidden"):
			actions.add("Open Settings and verify the provider API key or bearer credential.")
		case containsAny(text, "http 404", "not found"):
			actions.add("Verify the OpenAI-compatible base URL includes the provider's expected /v1 path.")
		case containsAny(text, "http 429", "rate limit", "too many requests"):
			actions.add("Provider is rate-limiting requests; wait or switch to another configured model/provider route.")
		case containsAny(text, "http 500", "http 502", "http 503", "http 504"):
			actions.add("Provider is unhealthy or overloaded; check runtime logs and retry after it recovers.")
		default:
			actions.add("Run the provider probe again after checking endpoint health, credentials, and model availability.")
		}
	}

	if containsAny(text, "configured model was not returned") {
		if ollama && strings.TrimSpace(config.Model) != "" {
			actions.add(fmt.Sprintf(`Run "ollama pull %s" or choose a model shown by "ollama list".`, config.Model))
		} else {
			actions.add("Select a model returned by the provider or install the configured model before running assistant tasks.")
		}
	}

	if result.ModelCount == 0 && result.OK {
		actions.add("Provider returned no models; install or expose at least one chat-capable model before using Ask or Agent.")
	}

	if result.Runtime != nil {
		runtimeText := strings.ToLower(result.Runtime.Message)
		switch {
		case strings.TrimSpace(result.Runtime.SelectedModel) != "" && !result.Runtime.SelectedModelLoaded:
			if ollama {
				actions.add(fmt.Sprintf(`Warm the selected model with "ollama run %s" or send one small request before long workflows.`, result.Runtime.SelectedModel))
			} else {
				actions.add("Load the selected model in your runtime or switch Settings to an already-loaded model.")
			}
		case containsAny(runtimeText, "no models are loaded"):
			actions.add("No runtime models are loaded yet; this is normal before first use, but installed models should still appear in the model list.")
		}
	}

	return actions.values
}

func isOllamaConfig(config Config, result ProbeResult) bool {
	text := strings.ToLower(strings.Join([]string{
		config.Provider,
		config.Protocol,
		config.BaseURL,
		result.Protocol,
		result.Endpoint,
	}, " "))
	return containsAny(text, "ollama", "localhost:11434", "127.0.0.1:11434")
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type guidanceAccumulator struct {
	values []string
	seen   map[string]struct{}
}

func (g *guidanceAccumulator) add(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	key := strings.ToLower(value)
	if g.seen == nil {
		g.seen = map[string]struct{}{}
	}
	if _, ok := g.seen[key]; ok {
		return
	}
	g.seen[key] = struct{}{}
	g.values = append(g.values, value)
}

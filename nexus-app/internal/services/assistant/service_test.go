package assistant

import (
	"context"
	"errors"
	"strings"
	"testing"

	"nexusdesk/internal/services/llm"
	settingssvc "nexusdesk/internal/services/settings"
	workspacesvc "nexusdesk/internal/services/workspace"
)

func TestAskStreamLoadsSettingsAndStreamsSelectedContext(t *testing.T) {
	store := fakeSettingsStore{settings: settingssvc.Settings{
		Provider:              "openai-compatible",
		BaseURL:               "http://provider.test/v1",
		Model:                 "model-a",
		ContextTokens:         2000,
		ResponseReserveTokens: 500,
	}}
	contextPacker := &fakeContextPacker{pack: workspacesvc.ContextPack{
		Label:       "context: README.md",
		Content:     "workspace context",
		SourcePaths: []string{"README.md"},
	}}
	client := &fakeStreamClient{message: "final answer", deltas: []string{"final ", "answer"}}
	service := NewWithDependencies(store, contextPacker, client)

	var streamed strings.Builder
	result, err := service.AskStream(context.Background(), Request{
		Prompt:        "Summarize",
		WorkspaceRoot: "C:/repo",
		SelectedPath:  "README.md",
	}, func(delta string) error {
		streamed.WriteString(delta)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message != "final answer" || streamed.String() != "final answer" {
		t.Fatalf("unexpected result=%q streamed=%q", result.Message, streamed.String())
	}
	if client.config.Model != "model-a" {
		t.Fatalf("settings were not passed to LLM client: %#v", client.config)
	}
	if client.request.ContextRelPath != "context: README.md" || client.request.ContextContent != "workspace context" {
		t.Fatalf("selected context was not attached: %#v", client.request)
	}
	if !strings.Contains(client.request.SystemPrompt, "Nexus Ask mode") {
		t.Fatalf("expected Ask-specific system prompt, got %q", client.request.SystemPrompt)
	}
	if len(contextPacker.paths) != 1 || contextPacker.paths[0] != "README.md" {
		t.Fatalf("unexpected context paths: %#v", contextPacker.paths)
	}
}

func TestAskStreamUsesPinnedContextPathsBeforeSelectedPath(t *testing.T) {
	store := fakeSettingsStore{settings: settingssvc.Defaults()}
	contextPacker := &fakeContextPacker{pack: workspacesvc.ContextPack{
		Label:       "context: 2 roots",
		Content:     "workspace context",
		SourcePaths: []string{"README.md", "docs/guide.md"},
	}}
	client := &fakeStreamClient{message: "ok", deltas: []string{"ok"}}
	service := NewWithDependencies(store, contextPacker, client)

	_, err := service.AskStream(context.Background(), Request{
		Prompt:        "Summarize",
		WorkspaceRoot: "C:/repo",
		SelectedPath:  "ignored.md",
		ContextPaths:  []string{"README.md", "docs", "README.md", " "},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(contextPacker.paths) != 2 || contextPacker.paths[0] != "README.md" || contextPacker.paths[1] != "docs" {
		t.Fatalf("expected deduplicated pinned context paths, got %#v", contextPacker.paths)
	}
}

func TestAskStreamCapsSelectedContextToBudget(t *testing.T) {
	store := fakeSettingsStore{settings: settingssvc.Settings{
		Provider:              "openai-compatible",
		BaseURL:               "http://provider.test/v1",
		Model:                 "model-a",
		ContextTokens:         20,
		ResponseReserveTokens: 10,
	}}
	contextPacker := &fakeContextPacker{pack: workspacesvc.ContextPack{
		Label:       "context: large.txt",
		Content:     strings.Repeat("a", 40) + "\n[context pack truncated]",
		SourcePaths: []string{"large.txt"},
		Truncated:   true,
	}}
	client := &fakeStreamClient{message: "ok", deltas: []string{"ok"}}
	service := NewWithDependencies(store, contextPacker, client)

	result, err := service.AskStream(context.Background(), Request{
		Prompt:        "Read",
		WorkspaceRoot: "C:/repo",
		SelectedPath:  "large.txt",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.ContextWarning == "" {
		t.Fatal("expected context warning")
	}
	if contextPacker.options.MaxBytes != 40 {
		t.Fatalf("expected model budget to be passed, got %d", contextPacker.options.MaxBytes)
	}
	if !strings.Contains(client.request.ContextContent, "[context pack truncated]") {
		t.Fatalf("expected capped context, got %q", client.request.ContextContent)
	}
}

func TestAskStreamSkipsBinarySelection(t *testing.T) {
	store := fakeSettingsStore{settings: settingssvc.Defaults()}
	contextPacker := &fakeContextPacker{err: errNoContext}
	client := &fakeStreamClient{message: "ok", deltas: []string{"ok"}}
	service := NewWithDependencies(store, contextPacker, client)

	result, err := service.AskStream(context.Background(), Request{
		Prompt:        "Describe",
		WorkspaceRoot: "C:/repo",
		SelectedPath:  "image.png",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.ContextWarning == "" {
		t.Fatal("expected non-text context warning")
	}
	if client.request.ContextContent != "" {
		t.Fatalf("binary context should not be attached: %#v", client.request)
	}
}

func TestAskStreamAppliesAssistantProfileWhenConfigured(t *testing.T) {
	store := fakeSettingsStore{settings: settingssvc.Defaults()}
	client := &fakeStreamClient{message: "ok", deltas: []string{"ok"}}
	service := NewWithDependencies(store, nil, client)
	service.SetProfileStore(fakeProfileStore{profile: Profile{
		Memory:          "Prefer compact answers.",
		ActiveProfileID: "reviewer",
		PromptProfiles:  DefaultProfile().PromptProfiles,
	}})

	_, err := service.AskStream(context.Background(), Request{Prompt: "Review this"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Active prompt profile: Reviewer", "Prefer compact answers.", "User request:", "Review this"} {
		if !strings.Contains(client.request.Prompt, expected) {
			t.Fatalf("expected profiled prompt to contain %q, got %q", expected, client.request.Prompt)
		}
	}
}

func TestAskStreamAppliesRequestedModelRoute(t *testing.T) {
	store := fakeSettingsStore{settings: settingssvc.Settings{
		Provider:              "ollama",
		Protocol:              settingssvc.ProtocolOllamaOpenAICompatible,
		BaseURL:               "http://localhost:11434/v1",
		Model:                 "global-model",
		ContextTokens:         32000,
		ResponseReserveTokens: 4000,
		ModelRoutes: []settingssvc.ModelRoute{
			{
				ID:                    settingssvc.RouteMainCoding,
				Label:                 "Main coding model",
				Model:                 "qwen3-coder:30b",
				ContextTokens:         131072,
				ResponseReserveTokens: 16384,
			},
		},
	}}
	client := &fakeStreamClient{message: "ok", deltas: []string{"ok"}}
	service := NewWithDependencies(store, nil, client)

	result, err := service.AskStream(context.Background(), Request{
		Prompt:       "Review diff",
		ModelRouteID: settingssvc.RouteMainCoding,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if client.config.Model != "qwen3-coder:30b" || client.config.ContextTokens != 131072 {
		t.Fatalf("expected routed model config, got %#v", client.config)
	}
	if result.ModelRouteID != settingssvc.RouteMainCoding || result.ModelRoute != "Main coding model" || result.RouteWarning != "" {
		t.Fatalf("expected route metadata in result, got %#v", result)
	}
}

func TestAskStreamFallsBackWhenRequestedModelRouteIsMissing(t *testing.T) {
	store := fakeSettingsStore{settings: settingssvc.Settings{
		Model:                 "global-model",
		ContextTokens:         32000,
		ResponseReserveTokens: 4000,
		ModelRoutes:           []settingssvc.ModelRoute{},
	}}
	client := &fakeStreamClient{message: "ok", deltas: []string{"ok"}}
	service := NewWithDependencies(store, nil, client)

	result, err := service.AskStream(context.Background(), Request{
		Prompt:       "Review diff",
		ModelRouteID: "missing-route",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if client.config.Model != "global-model" {
		t.Fatalf("expected global model fallback, got %#v", client.config)
	}
	if result.RouteWarning == "" || !strings.Contains(result.RouteWarning, "using the global model") {
		t.Fatalf("expected route warning, got %#v", result)
	}
}

type fakeSettingsStore struct {
	settings settingssvc.Settings
	err      error
}

func (s fakeSettingsStore) Load() (settingssvc.Settings, error) {
	return s.settings, s.err
}

type fakeContextPacker struct {
	pack    workspacesvc.ContextPack
	options workspacesvc.ContextPackOptions
	paths   []string
	err     error
}

func (p *fakeContextPacker) BuildContextPack(_ string, paths []string, options workspacesvc.ContextPackOptions) (workspacesvc.ContextPack, error) {
	p.paths = append([]string{}, paths...)
	p.options = options
	return p.pack, p.err
}

var errNoContext = errors.New("context paths did not contain previewable text files")

type fakeProfileStore struct {
	profile Profile
	err     error
}

func (s fakeProfileStore) Get() (Profile, error) {
	return s.profile, s.err
}

type fakeStreamClient struct {
	config  llm.Config
	request llm.ChatRequest
	message string
	deltas  []string
}

func (c *fakeStreamClient) ChatStream(_ context.Context, config llm.Config, request llm.ChatRequest, onDelta func(string) error) (llm.ChatResult, error) {
	c.config = config
	c.request = request
	for _, delta := range c.deltas {
		if onDelta != nil {
			if err := onDelta(delta); err != nil {
				return llm.ChatResult{}, err
			}
		}
	}
	return llm.ChatResult{
		Message:        c.message,
		Model:          config.Model,
		Endpoint:       "http://provider.test/v1/chat/completions",
		ContextRelPath: request.ContextRelPath,
		SourcePaths:    append([]string{}, request.SourcePaths...),
	}, nil
}

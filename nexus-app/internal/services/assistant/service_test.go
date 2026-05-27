package assistant

import (
	"context"
	"strings"
	"testing"

	"nexusdesk/internal/domain"
	"nexusdesk/internal/services/llm"
	settingssvc "nexusdesk/internal/services/settings"
)

func TestAskStreamLoadsSettingsAndStreamsSelectedContext(t *testing.T) {
	store := fakeSettingsStore{settings: settingssvc.Settings{
		Provider:              "openai-compatible",
		BaseURL:               "http://provider.test/v1",
		Model:                 "model-a",
		ContextTokens:         2000,
		ResponseReserveTokens: 500,
	}}
	previewer := fakePreviewer{preview: domain.FilePreview{
		RelPath: "README.md",
		Kind:    domain.PreviewText,
		Text:    "workspace context",
	}}
	client := &fakeStreamClient{message: "final answer", deltas: []string{"final ", "answer"}}
	service := NewWithDependencies(store, previewer, client)

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
	if client.request.ContextRelPath != "README.md" || client.request.ContextContent != "workspace context" {
		t.Fatalf("selected context was not attached: %#v", client.request)
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
	previewer := fakePreviewer{preview: domain.FilePreview{
		RelPath: "large.txt",
		Kind:    domain.PreviewText,
		Text:    strings.Repeat("a", 100),
	}}
	client := &fakeStreamClient{message: "ok", deltas: []string{"ok"}}
	service := NewWithDependencies(store, previewer, client)

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
	if !strings.Contains(client.request.ContextContent, "[context truncated]") {
		t.Fatalf("expected capped context, got %q", client.request.ContextContent)
	}
}

func TestAskStreamSkipsBinarySelection(t *testing.T) {
	store := fakeSettingsStore{settings: settingssvc.Defaults()}
	previewer := fakePreviewer{preview: domain.FilePreview{RelPath: "image.png", Kind: domain.PreviewImage}}
	client := &fakeStreamClient{message: "ok", deltas: []string{"ok"}}
	service := NewWithDependencies(store, previewer, client)

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

type fakeSettingsStore struct {
	settings settingssvc.Settings
	err      error
}

func (s fakeSettingsStore) Load() (settingssvc.Settings, error) {
	return s.settings, s.err
}

type fakePreviewer struct {
	preview domain.FilePreview
	err     error
}

func (p fakePreviewer) PreviewFile(string, string) (domain.FilePreview, error) {
	return p.preview, p.err
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

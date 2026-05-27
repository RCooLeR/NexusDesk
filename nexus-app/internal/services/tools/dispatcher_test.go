package tools

import (
	"context"
	"strings"
	"testing"

	"nexusdesk/internal/services/agent"
)

func TestDispatcherExecutesRegisteredTool(t *testing.T) {
	dispatcher := NewDispatcher(Tool{
		Descriptor: agent.ToolDescriptor{Name: "hello", Description: "Say hello", Risk: "low"},
		Handler: func(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
			return agent.ToolResult{Observation: "hello " + call.Args["name"]}, nil
		},
	})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "hello", Args: map[string]string{"name": "Nexus"}}, agent.Request{})
	if err != nil {
		t.Fatalf("ExecuteTool returned error: %v", err)
	}
	if result.Name != "hello" || result.Risk != "low" || result.Observation != "hello Nexus" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestDispatcherReportsUnknownTool(t *testing.T) {
	dispatcher := NewDispatcher()
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "missing"}, agent.Request{})
	if err == nil || !strings.Contains(result.Error, "not registered") {
		t.Fatalf("expected unknown tool error, got result=%#v err=%v", result, err)
	}
}

func TestToolDescriptorsAreSorted(t *testing.T) {
	dispatcher := NewDispatcher(
		Tool{Descriptor: agent.ToolDescriptor{Name: "zeta"}, Handler: noopHandler},
		Tool{Descriptor: agent.ToolDescriptor{Name: "alpha"}, Handler: noopHandler},
	)
	descriptors := dispatcher.ToolDescriptors()
	if len(descriptors) != 2 || descriptors[0].Name != "alpha" || descriptors[1].Name != "zeta" {
		t.Fatalf("unexpected descriptors: %#v", descriptors)
	}
}

func noopHandler(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	return agent.ToolResult{}, nil
}

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

func TestDispatcherRequestsPerCallApprovalForHighRiskTool(t *testing.T) {
	called := false
	dispatcher := NewDispatcher(Tool{
		Descriptor: agent.ToolDescriptor{Name: "write_file", Description: "Write file", Risk: "high"},
		Handler: func(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
			if !request.ApproveWrites {
				t.Fatal("expected per-call approval to grant this write call")
			}
			return agent.ToolResult{Name: call.Name, Mutated: true}, nil
		},
	})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "write_file"}, agent.Request{
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool {
			called = true
			return request.Name == "write_file" && request.Risk == "high"
		},
	})
	if err != nil || !result.Mutated || !called {
		t.Fatalf("expected approved high-risk call, result=%#v err=%v called=%t", result, err, called)
	}
}

func TestDispatcherBlocksDeniedPerCallApproval(t *testing.T) {
	dispatcher := NewDispatcher(Tool{
		Descriptor: agent.ToolDescriptor{Name: "write_file", Risk: "high"},
		Handler: func(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
			t.Fatal("denied tool should not execute")
			return agent.ToolResult{}, nil
		},
	})
	result, err := dispatcher.ExecuteTool(context.Background(), agent.ToolCall{Name: "write_file"}, agent.Request{
		ApproveTool: func(ctx context.Context, request agent.ToolApprovalRequest) bool { return false },
	})
	if err == nil || !strings.Contains(result.Error, "per-call approval") {
		t.Fatalf("expected per-call approval denial, result=%#v err=%v", result, err)
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

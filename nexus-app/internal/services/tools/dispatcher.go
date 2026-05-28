package tools

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"nexusdesk/internal/services/agent"
)

type Dispatcher struct {
	tools map[string]Tool
}

func NewDispatcher(tools ...Tool) *Dispatcher {
	dispatcher := &Dispatcher{tools: map[string]Tool{}}
	for _, tool := range tools {
		dispatcher.Register(tool)
	}
	return dispatcher
}

func (d *Dispatcher) Register(tool Tool) {
	name := strings.TrimSpace(tool.Descriptor.Name)
	if name == "" || tool.Handler == nil {
		return
	}
	tool.Descriptor.Name = name
	d.tools[name] = tool
}

func (d *Dispatcher) ExecuteTool(ctx context.Context, call agent.ToolCall, request agent.Request) (agent.ToolResult, error) {
	name := strings.TrimSpace(call.Name)
	tool, ok := d.tools[name]
	if !ok {
		result := agent.ToolResult{Name: name, Args: call.Args, Risk: "unknown", Error: "tool is not registered"}
		return result, fmt.Errorf("tool %q is not registered", name)
	}
	if toolRequiresPerCallApproval(tool, request) {
		if request.ApproveTool == nil || !request.ApproveTool(ctx, agent.ToolApprovalRequest{
			Name:        name,
			Args:        call.Args,
			Risk:        tool.Descriptor.Risk,
			Description: tool.Descriptor.Description,
		}) {
			err := errors.New("per-call approval was denied for high-risk agent tool")
			return agent.ToolResult{Name: name, Args: call.Args, Risk: tool.Descriptor.Risk, Observation: err.Error(), Error: err.Error()}, err
		}
		request = requestWithPerCallApproval(request, name)
	}
	result, err := tool.Handler(ctx, call, request)
	if result.Name == "" {
		result.Name = name
	}
	if result.Args == nil {
		result.Args = call.Args
	}
	if result.Risk == "" {
		result.Risk = tool.Descriptor.Risk
	}
	return result, err
}

func toolRequiresPerCallApproval(tool Tool, request agent.Request) bool {
	if strings.ToLower(strings.TrimSpace(tool.Descriptor.Risk)) != "high" {
		return false
	}
	switch tool.Descriptor.Name {
	case "run_task":
		return !request.ApproveShell
	default:
		return !request.ApproveWrites
	}
}

func requestWithPerCallApproval(request agent.Request, toolName string) agent.Request {
	if toolName == "run_task" {
		request.ApproveShell = true
		return request
	}
	request.ApproveWrites = true
	return request
}

func (d *Dispatcher) ToolDescriptors() []agent.ToolDescriptor {
	descriptors := make([]agent.ToolDescriptor, 0, len(d.tools))
	for _, tool := range d.tools {
		descriptors = append(descriptors, tool.Descriptor)
	}
	sort.Slice(descriptors, func(left int, right int) bool {
		return descriptors[left].Name < descriptors[right].Name
	})
	return descriptors
}

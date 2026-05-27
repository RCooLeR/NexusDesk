package tools

import (
	"context"
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

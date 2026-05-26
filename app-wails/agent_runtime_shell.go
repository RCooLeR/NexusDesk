package main

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"NexusAugenticStudio/internal/agent"
	"NexusAugenticStudio/internal/processutil"
)

func (a *App) agentExecuteShell(ctx context.Context, root string, call agent.ToolCall, request agent.RunRequest) (agent.ToolCall, error) {
	command := strings.TrimSpace(call.Arguments["command"])
	if command == "" {
		call.Error = "shell command is required"
		return call, errors.New(call.Error)
	}
	if !request.AllowShellCommands || !request.ApproveHighImpact {
		call.Observation = "Approval required before running shell command inside workspace: " + command
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	executable, args, ok := parseAllowedAgentShellCommand(command)
	if !ok {
		call.Observation = "Shell command blocked by workspace sandbox policy: " + command
		call.Error = "command escapes workspace sandbox"
		return call, errors.New(call.Error)
	}

	shellCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(shellCtx, executable, args...)
	processutil.ConfigureHiddenCommand(cmd)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	call.Observation = limitAgentOutput(string(output), maxAgentShellOutputBytes)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	a.recordApproval("agent.shell", command, "high", "Shell command executed inside workspace.")
	return call, nil
}

func parseAllowedAgentShellCommand(command string) (string, []string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(command))
	if normalized == "" {
		return "", nil, false
	}
	if strings.ContainsAny(command, "&|;<>`$%*?{}[]()") || strings.Contains(command, "\n") || strings.Contains(command, "\r") {
		return "", nil, false
	}
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", nil, false
	}
	executable := strings.ToLower(parts[0])
	args := parts[1:]
	if runtime.GOOS == "windows" {
		executable = strings.TrimSuffix(executable, ".exe")
		executable = strings.TrimSuffix(executable, ".cmd")
		executable = strings.TrimSuffix(executable, ".bat")
	}
	if !isAllowedAgentExecutable(executable, args) {
		return "", nil, false
	}
	for _, arg := range args {
		if !isWorkspaceRelativeShellArg(arg) {
			return "", nil, false
		}
	}
	return parts[0], args, true
}

func isAllowedAgentExecutable(executable string, args []string) bool {
	switch executable {
	case "go", "node", "python", "python3":
		return true
	case "npm":
		return len(args) > 0 && (args[0] == "run" || args[0] == "test" || args[0] == "install")
	case "npx":
		return len(args) > 0
	case "docker":
		return len(args) > 0 && (args[0] == "ps" || args[0] == "logs" || args[0] == "compose")
	case "git":
		return len(args) > 0 && isAllowedAgentGitSubcommand(args[0])
	default:
		return false
	}
}

func isAllowedAgentGitSubcommand(subcommand string) bool {
	switch subcommand {
	case "status", "diff", "log", "show", "branch", "rev-parse", "ls-files", "grep":
		return true
	default:
		return false
	}
}

func isWorkspaceRelativeShellArg(arg string) bool {
	trimmed := strings.TrimSpace(strings.Trim(arg, `"'`))
	if trimmed == "" {
		return true
	}
	normalized := filepath.ToSlash(strings.ToLower(trimmed))
	if normalized == "." || strings.HasPrefix(normalized, "-") {
		return true
	}
	if normalized == "./..." || normalized == "..." {
		return true
	}
	if strings.Contains(normalized, "..") || strings.HasPrefix(normalized, "~") {
		return false
	}
	if filepath.IsAbs(trimmed) || strings.HasPrefix(normalized, "/") || strings.HasPrefix(normalized, `\\`) {
		return false
	}
	if len(normalized) >= 2 && normalized[1] == ':' {
		return false
	}
	return true
}

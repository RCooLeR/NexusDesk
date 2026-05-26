package main

import (
	"errors"
	"strings"

	"NexusAugenticStudio/internal/agent"
	"NexusAugenticStudio/internal/workspace"
)

func (a *App) agentWriteFile(root string, call agent.ToolCall, approved bool) (agent.ToolCall, error) {
	request := workspace.FileWriteRequest{
		RelPath:  cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"])),
		Content:  call.Arguments["content"],
		Encoding: strings.TrimSpace(call.Arguments["encoding"]),
	}
	proposal, err := workspace.PreviewFileWrite(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	if !approved {
		call.Observation = "Approval required before writing file. Proposed diff:\n" + proposal.Diff
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	rollback, err := workspace.PrepareRollback(root, "agent.write_file", request.RelPath, []string{request.RelPath})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	applied, err := workspace.ApplyFileWrite(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	rollback, err = workspace.CommitRollback(root, rollback)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	a.recordApproval("agent.write_file", applied.RelPath, "high", applied.Message)
	call.Observation = applied.Message + "\nRollback: " + rollback.ID + "\n" + proposal.Diff
	return call, nil
}

func (a *App) agentAppendFile(root string, call agent.ToolCall, approved bool) (agent.ToolCall, error) {
	relPath := cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"]))
	content := call.Arguments["content"]
	if content == "" {
		call.Error = "append content is required"
		return call, errors.New(call.Error)
	}
	encoding := strings.TrimSpace(call.Arguments["encoding"])
	if encoding == "" {
		if preview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: 64 * 1024}); err == nil && preview.Encoding != "" {
			encoding = preview.Encoding
		}
	}
	request := workspace.FileWriteRequest{RelPath: relPath, Content: content, Encoding: encoding}
	proposal, err := workspace.PreviewFileAppend(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	if !approved {
		call.Observation = "Approval required before appending file. Proposed diff:\n" + proposal.Diff
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	rollback, err := workspace.PrepareRollback(root, "agent.append_file", request.RelPath, []string{request.RelPath})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	applied, err := workspace.ApplyFileAppend(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	rollback, err = workspace.CommitRollback(root, rollback)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	a.recordApproval("agent.append_file", applied.RelPath, "high", applied.Message)
	call.Observation = applied.Message + "\nRollback: " + rollback.ID + "\n" + proposal.Diff
	return call, nil
}

func (a *App) agentWriteBinaryFile(root string, call agent.ToolCall, approved bool) (agent.ToolCall, error) {
	request := workspace.BinaryFileWriteRequest{
		RelPath:       cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"])),
		Base64Content: firstNonEmpty(call.Arguments["base64Content"], call.Arguments["contentBase64"], call.Arguments["base64"]),
		ContentType:   strings.TrimSpace(call.Arguments["contentType"]),
	}
	proposal, err := workspace.PreviewBinaryFileWrite(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	if !approved {
		call.Observation = "Approval required before writing binary file. Proposed binary write:\n" + proposal.Diff
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	rollback, err := workspace.PrepareRollback(root, "agent.write_binary_file", request.RelPath, []string{request.RelPath})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	applied, err := workspace.ApplyBinaryFileWrite(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	rollback, err = workspace.CommitRollback(root, rollback)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	a.recordApproval("agent.write_binary_file", applied.RelPath, "high", applied.Message)
	call.Observation = applied.Message + "\nRollback: " + rollback.ID + "\n" + proposal.Diff
	return call, nil
}

func (a *App) agentApplyPatch(root string, call agent.ToolCall, approved bool) (agent.ToolCall, error) {
	request := workspace.UnifiedPatchRequest{
		Patch: firstNonEmpty(call.Arguments["patch"], call.Arguments["unifiedDiff"], call.Arguments["diff"]),
	}
	proposal, err := workspace.PreviewUnifiedPatch(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	if !approved {
		call.Observation = "Approval required before applying patch. Proposed patch:\n" + unifiedPatchProposalSummary(proposal, true)
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	rollbackPaths := unifiedPatchRollbackPaths(proposal)
	rollback, err := workspace.PrepareRollback(root, "agent.apply_patch", "workspace", rollbackPaths)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	applied, err := workspace.ApplyUnifiedPatch(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	rollback, err = workspace.CommitRollback(root, rollback)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	a.recordApproval("agent.apply_patch", "workspace", "high", applied.Message)
	call.Observation = applied.Message + "\nRollback: " + rollback.ID + "\n" + unifiedPatchProposalSummary(proposal, true)
	return call, nil
}

func (a *App) agentCopyFile(root string, call agent.ToolCall, approved bool) (agent.ToolCall, error) {
	request := workspace.FileCopyRequest{
		SourceRelPath: cleanAgentRelPath(firstNonEmpty(call.Arguments["sourceRelPath"], call.Arguments["source"], call.Arguments["from"])),
		TargetRelPath: cleanAgentRelPath(firstNonEmpty(call.Arguments["targetRelPath"], call.Arguments["target"], call.Arguments["to"])),
	}
	proposal, err := workspace.PreviewFileCopy(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	if !approved {
		call.Observation = "Approval required before copying file. Proposed copy: " + proposal.Message
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	rollback, err := workspace.PrepareRollback(root, "agent.copy_file", request.TargetRelPath, []string{request.TargetRelPath})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	applied, err := workspace.ApplyFileCopy(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	rollback, err = workspace.CommitRollback(root, rollback)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	a.recordApproval("agent.copy_file", applied.TargetRelPath, "high", applied.Message)
	call.Observation = applied.Message + "\nRollback: " + rollback.ID
	return call, nil
}

func (a *App) agentMoveFile(root string, call agent.ToolCall, approved bool) (agent.ToolCall, error) {
	request := workspace.FileMoveRequest{
		SourceRelPath: cleanAgentRelPath(firstNonEmpty(call.Arguments["sourceRelPath"], call.Arguments["source"], call.Arguments["from"])),
		TargetRelPath: cleanAgentRelPath(firstNonEmpty(call.Arguments["targetRelPath"], call.Arguments["target"], call.Arguments["to"])),
	}
	proposal, err := workspace.PreviewFileMove(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	if !approved {
		call.Observation = "Approval required before moving file. Proposed move: " + proposal.Message
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	rollback, err := workspace.PrepareRollback(root, "agent.move_file", request.TargetRelPath, []string{request.SourceRelPath, request.TargetRelPath})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	applied, err := workspace.ApplyFileMove(root, request)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	rollback, err = workspace.CommitRollback(root, rollback)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	a.recordApproval("agent.move_file", applied.TargetRelPath, "high", applied.Message)
	call.Observation = applied.Message + "\nRollback: " + rollback.ID
	return call, nil
}

func (a *App) agentDeleteFile(root string, call agent.ToolCall, approved bool) (agent.ToolCall, error) {
	relPath := cleanAgentRelPath(firstNonEmpty(call.Arguments["relPath"], call.Arguments["path"], call.Arguments["target"]))
	proposal, err := workspace.PreviewFileDelete(root, relPath)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	if !approved {
		call.Observation = "Approval required before deleting file. Proposed delete:\n" + firstNonEmpty(proposal.Diff, proposal.Message)
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	rollback, err := workspace.PrepareRollback(root, "agent.delete_file", relPath, []string{relPath})
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	applied, err := workspace.ApplyFileDelete(root, relPath)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	rollback, err = workspace.CommitRollback(root, rollback)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	a.recordApproval("agent.delete_file", applied.RelPath, "high", applied.Message)
	call.Observation = applied.Message + "\nRollback: " + rollback.ID
	return call, nil
}

func (a *App) agentListRollbacks(root string, call agent.ToolCall) (agent.ToolCall, error) {
	items, err := workspace.ListRollbacks(root)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	call.Observation = formatRollbackListObservation(items)
	return call, nil
}

func (a *App) agentRollbackFileMutation(root string, call agent.ToolCall, approved bool) (agent.ToolCall, error) {
	id := strings.TrimSpace(firstNonEmpty(call.Arguments["id"], call.Arguments["rollbackId"]))
	if id == "" {
		call.Error = "rollback id is required"
		return call, errors.New(call.Error)
	}
	if !approved {
		call.Observation = "Approval required before applying rollback: " + id
		call.Error = "approval required"
		return call, errors.New(call.Error)
	}
	result, err := workspace.ApplyRollback(root, id)
	if err != nil {
		call.Error = err.Error()
		return call, err
	}
	a.recordApproval("agent.rollback_file_mutation", result.ID, "high", result.Message)
	call.Observation = formatRollbackApplyObservation(result)
	return call, nil
}

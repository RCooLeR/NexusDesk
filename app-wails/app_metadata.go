package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"NexusAugenticStudio/internal/agenttools"
	"NexusAugenticStudio/internal/appmeta"
	"NexusAugenticStudio/internal/approval"
	"NexusAugenticStudio/internal/artifact"
	"NexusAugenticStudio/internal/dataset"
	"NexusAugenticStudio/internal/storage"
)

func (a *App) syncPreparedMetadataStore(root string) {
	if root == "" || !appmeta.Exists(root) {
		return
	}
	_, _ = a.mirrorMetadataStore(root, false)
}

func (a *App) mirrorMetadataStore(root string, create bool) (appmeta.SQLiteStatus, error) {
	if !create && !appmeta.Exists(root) {
		return appmeta.SQLiteStatus{}, nil
	}
	data, err := a.metadataMirrorData(root)
	if err != nil {
		return appmeta.SQLiteStatus{}, err
	}
	return appmeta.Mirror(root, data)
}

func (a *App) metadataMirrorData(root string) (appmeta.MirrorData, error) {
	chats, err := a.chatStore.List(root)
	if err != nil {
		return appmeta.MirrorData{}, err
	}
	approvals, err := approval.List(root)
	if err != nil {
		return appmeta.MirrorData{}, err
	}
	artifacts, err := artifact.List(root)
	if err != nil {
		return appmeta.MirrorData{}, err
	}
	toolRuns, err := agenttools.List(root)
	if err != nil {
		return appmeta.MirrorData{}, err
	}

	data := appmeta.MirrorData{
		Chats:     make([]appmeta.ChatMirror, 0, len(chats)),
		Approvals: make([]appmeta.ApprovalMirror, 0, len(approvals)),
		Artifacts: make([]appmeta.ArtifactMirror, 0, len(artifacts)),
		ToolRuns:  make([]appmeta.ToolRunMirror, 0, len(toolRuns)),
	}
	for index, message := range chats {
		data.Chats = append(data.Chats, appmeta.ChatMirror{
			ID:             fmt.Sprintf("chat-%03d-%s", index, hashForID(message.Role+message.CreatedAt+message.Content)),
			Role:           message.Role,
			Content:        message.Content,
			ContextRelPath: message.ContextRelPath,
			SourcePaths:    message.SourcePaths,
			CreatedAt:      message.CreatedAt,
		})
	}
	for _, record := range approvals {
		data.Approvals = append(data.Approvals, appmeta.ApprovalMirror{
			ID:        record.ID,
			Action:    record.Action,
			Target:    record.Target,
			Risk:      record.Risk,
			Decision:  record.Decision,
			Message:   record.Message,
			CreatedAt: record.CreatedAt,
		})
	}
	for _, item := range artifacts {
		metadata, _ := artifact.Metadata(root, item.RelPath)
		payload, _ := json.Marshal(metadata)
		data.Artifacts = append(data.Artifacts, appmeta.ArtifactMirror{
			ID:             "artifact-" + hashForID(item.RelPath),
			RelPath:        item.RelPath,
			Kind:           item.Kind,
			Title:          metadata.Title,
			Source:         metadata.Source,
			ContextRelPath: metadata.ContextRelPath,
			Metadata:       payload,
			CreatedAt:      fallbackInput(metadata.CreatedAt, item.ModifiedAt),
		})
	}
	for _, run := range toolRuns {
		inputs, _ := json.Marshal(run.Inputs)
		data.ToolRuns = append(data.ToolRuns, appmeta.ToolRunMirror{
			ID:            run.ID,
			ToolName:      run.ToolName,
			Target:        run.Target,
			Risk:          run.Risk,
			Status:        run.Status,
			Mode:          run.Mode,
			ApprovalID:    run.ApprovalID,
			Inputs:        inputs,
			OutputSummary: run.OutputSummary,
			Error:         run.Error,
			StartedAt:     run.StartedAt,
			CompletedAt:   run.CompletedAt,
			DurationMs:    run.DurationMs,
		})
	}
	return data, nil
}

func (a *App) listArtifactsFromMetadata(root string) ([]artifact.WorkspaceArtifact, error) {
	items, err := appmeta.ListArtifacts(root)
	if err != nil {
		return nil, err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	artifacts := make([]artifact.WorkspaceArtifact, 0, len(items))
	for _, item := range items {
		path := filepath.Join(absRoot, filepath.FromSlash(item.RelPath))
		info, statErr := os.Stat(path)
		if statErr != nil {
			continue
		}
		var metadata artifact.ArtifactMetadata
		_ = json.Unmarshal(item.Metadata, &metadata)
		source := metadata.Source
		if source == "" {
			source = item.Source
		}
		artifacts = append(artifacts, artifact.WorkspaceArtifact{
			RelPath:    item.RelPath,
			Name:       filepath.Base(item.RelPath),
			Path:       path,
			Kind:       item.Kind,
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
			Source:     source,
			Summary:    artifactSummaryFromMetadata(metadata),
			Model:      metadata.Model,
		})
	}
	return artifacts, nil
}

func (a *App) persistArtifactMetadata(root string, relPath string) {
	if root == "" || relPath == "" || !appmeta.Exists(root) {
		return
	}
	item, err := artifact.Metadata(root, relPath)
	if err != nil {
		return
	}
	payload, _ := json.Marshal(item)
	_ = appmeta.UpsertArtifact(root, appmeta.ArtifactMirror{
		ID:             "artifact-" + hashForID(relPath),
		RelPath:        relPath,
		Kind:           item.Kind,
		Title:          item.Title,
		Source:         item.Source,
		ContextRelPath: item.ContextRelPath,
		Metadata:       payload,
		CreatedAt:      item.CreatedAt,
	})
}

func (a *App) recordDatasetDependency(root string, relPath string, kind string, query string, target string, artifactRelPath string) {
	if root == "" || relPath == "" || !appmeta.Exists(root) {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_ = appmeta.RecordDatasetDependency(root, appmeta.DatasetDependency{
		ID:          "dataset-dependency-" + hashForID(relPath+kind+query+target+artifactRelPath+now),
		RelPath:     filepath.ToSlash(relPath),
		Kind:        kind,
		Target:      target,
		Query:       query,
		Artifact:    artifactRelPath,
		CreatedAt:   now,
		LastRefresh: now,
	})
}

func (a *App) recordSQLRun(root string, relPath string, sqlText string, engine string, rows int, artifactRelPath string, status string, message string) {
	if root == "" || relPath == "" || !appmeta.Exists(root) {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_ = appmeta.AppendSQLRun(root, appmeta.SQLRun{
		ID:        "sql-run-" + hashForID(relPath+sqlText+now),
		RelPath:   filepath.ToSlash(relPath),
		SQL:       sqlText,
		Engine:    fallbackInput(engine, "unknown"),
		Rows:      rows,
		Artifact:  artifactRelPath,
		Status:    status,
		Message:   message,
		CreatedAt: now,
	})
}

func chatMirrorFromMessage(message storage.ChatMessage, id string) appmeta.ChatMirror {
	return appmeta.ChatMirror{
		ID:             id,
		Role:           message.Role,
		Content:        message.Content,
		ContextRelPath: message.ContextRelPath,
		SourcePaths:    message.SourcePaths,
		CreatedAt:      message.CreatedAt,
	}
}

func chatsFromMirror(items []appmeta.ChatMirror) []storage.ChatMessage {
	messages := make([]storage.ChatMessage, 0, len(items))
	for _, item := range items {
		messages = append(messages, storage.ChatMessage{
			Role:           item.Role,
			Content:        item.Content,
			ContextRelPath: item.ContextRelPath,
			SourcePaths:    item.SourcePaths,
			CreatedAt:      item.CreatedAt,
		})
	}
	return messages
}

func approvalsFromMirror(items []appmeta.ApprovalMirror) []approval.Record {
	records := make([]approval.Record, 0, len(items))
	for _, item := range items {
		records = append(records, approval.Record{
			ID:        item.ID,
			Action:    item.Action,
			Target:    item.Target,
			Risk:      item.Risk,
			Decision:  item.Decision,
			Message:   item.Message,
			CreatedAt: item.CreatedAt,
		})
	}
	return records
}

func toolRunsFromMirror(items []appmeta.ToolRunMirror) []agenttools.RunRecord {
	records := make([]agenttools.RunRecord, 0, len(items))
	for _, item := range items {
		inputs := map[string]string{}
		_ = json.Unmarshal(item.Inputs, &inputs)
		title := item.ToolName
		requiresApproval := false
		if descriptor, ok := agenttools.Find(item.ToolName); ok {
			title = descriptor.Title
			requiresApproval = descriptor.RequiresApproval
		}
		records = append(records, agenttools.RunRecord{
			ID:               item.ID,
			ToolName:         item.ToolName,
			Title:            title,
			Target:           item.Target,
			Risk:             item.Risk,
			RequiresApproval: requiresApproval,
			Status:           item.Status,
			Mode:             item.Mode,
			Inputs:           inputs,
			OutputSummary:    item.OutputSummary,
			Error:            item.Error,
			ApprovalID:       item.ApprovalID,
			StartedAt:        item.StartedAt,
			CompletedAt:      item.CompletedAt,
			DurationMs:       item.DurationMs,
		})
	}
	return records
}

func artifactSummaryFromMetadata(metadata artifact.ArtifactMetadata) string {
	if metadata.Title != "" {
		return metadata.Title
	}
	if len(metadata.SourcePaths) == 1 {
		return metadata.SourcePaths[0]
	}
	if len(metadata.SourcePaths) > 1 {
		return fmt.Sprintf("%d source paths", len(metadata.SourcePaths))
	}
	return metadata.Source
}

func (a *App) datasetViews(root string) []appmeta.DatasetView {
	profiles, err := dataset.List(root)
	if err != nil {
		return []appmeta.DatasetView{}
	}
	views := []appmeta.DatasetView{}
	for _, profile := range profiles {
		columns := []string{}
		for _, column := range profile.Profiles {
			columns = append(columns, column.Name)
		}
		name := strings.TrimSuffix(filepath.Base(profile.RelPath), filepath.Ext(profile.RelPath))
		if name == "" {
			name = "dataset"
		}
		views = append(views, appmeta.DatasetView{
			Name:    name,
			RelPath: profile.RelPath,
			Engine:  "duckdb view / csv fallback",
			Columns: columns,
			Rows:    profile.Rows,
			Message: fmt.Sprintf("%s has %d columns and is addressable as dataset or %s in SQL.", profile.RelPath, profile.Columns, name),
		})
	}
	return views
}

func hashForID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 24 {
		value = value[:24]
	}
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, ":", "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "item"
	}
	return value
}

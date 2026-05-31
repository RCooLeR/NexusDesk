package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nexusdesk/internal/buildinfo"
	artifactsSvc "nexusdesk/internal/services/artifacts"
	datasetsSvc "nexusdesk/internal/services/datasets"
	issuereportSvc "nexusdesk/internal/services/issuereport"
	settingsSvc "nexusdesk/internal/services/settings"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

type smokeReport struct {
	App       buildinfo.Info `json:"app"`
	Workspace string         `json:"workspace"`
	Checks    []smokeCheck   `json:"checks"`
}

type smokeCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func RunSmokeCheck(root string, writer io.Writer) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return errors.New("smoke workspace root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(absRoot, 0o755); err != nil {
		return err
	}

	report := smokeReport{
		App:       buildinfo.Current(),
		Workspace: absRoot,
		Checks:    []smokeCheck{},
	}
	addCheck := func(name string, detail string) {
		report.Checks = append(report.Checks, smokeCheck{Name: name, Status: "ok", Detail: detail})
	}

	workspace := workspaceSvc.New()
	if err := writeSmokeFixtures(absRoot); err != nil {
		return err
	}
	opened, err := workspace.Open(absRoot)
	if err != nil {
		return fmt.Errorf("open workspace: %w", err)
	}
	addCheck("workspace-open", opened.Name)

	preview, err := workspace.PreviewFile(absRoot, "smoke-source.md")
	if err != nil {
		return fmt.Errorf("preview smoke source: %w", err)
	}
	if !strings.Contains(preview.Text, "platform-smoke-token") || preview.Truncated {
		return errors.New("preview did not expose expected smoke source text")
	}
	addCheck("file-preview", preview.RelPath)

	results, err := workspace.Search(absRoot, "platform-smoke-token", workspaceSvc.SearchOptions{MaxResults: 10})
	if err != nil {
		return fmt.Errorf("search smoke source: %w", err)
	}
	if len(results) == 0 {
		return errors.New("search did not find smoke token")
	}
	addCheck("workspace-search", fmt.Sprintf("%d result(s)", len(results)))

	writeResult, err := workspace.ApplyFileWrite(absRoot, workspaceSvc.FileWriteRequest{
		RelPath: "notes/smoke-edit.txt",
		Content: "temporary edit from packaged smoke\n",
	})
	if err != nil {
		return fmt.Errorf("apply smoke edit: %w", err)
	}
	if writeResult.RollbackID == "" {
		return errors.New("smoke edit did not create rollback")
	}
	if _, err := workspace.ApplyRollback(absRoot, writeResult.RollbackID); err != nil {
		return fmt.Errorf("rollback smoke edit: %w", err)
	}
	if _, err := os.Stat(filepath.Join(absRoot, "notes", "smoke-edit.txt")); !os.IsNotExist(err) {
		return errors.New("rollback did not remove smoke edit")
	}
	addCheck("edit-save-revert", writeResult.RollbackID)

	settingsPath := filepath.Join(absRoot, ".nexusdesk", "smoke", "settings.json")
	settingsStore := settingsSvc.NewFileStore(settingsPath)
	settings := settingsSvc.SettingsForSelectedModel(settingsSvc.Defaults(), "qwen3:8b")
	if err := settingsStore.Save(settings); err != nil {
		return fmt.Errorf("save smoke provider settings: %w", err)
	}
	loadedSettings, err := settingsStore.LoadForDisplay()
	if err != nil {
		return fmt.Errorf("load smoke provider settings: %w", err)
	}
	if loadedSettings.Provider == "" || loadedSettings.BaseURL == "" || loadedSettings.Model != "qwen3:8b" {
		return errors.New("smoke provider settings did not round-trip")
	}
	addCheck("assistant-settings", loadedSettings.Provider+"/"+loadedSettings.Model)

	datasets := datasetsSvc.New(workspace)
	profile, err := datasets.Profile(absRoot, "data/smoke.csv")
	if err != nil {
		return fmt.Errorf("profile smoke dataset: %w", err)
	}
	if profile.Format != "CSV" || len(profile.Columns) < 2 {
		return errors.New("smoke dataset profile did not include expected CSV columns")
	}
	addCheck("dataset-profile", fmt.Sprintf("%s %d column(s)", profile.Format, len(profile.Columns)))

	artifactStore, err := artifactsSvc.NewStore(absRoot)
	if err != nil {
		return fmt.Errorf("open artifact store: %w", err)
	}
	artifact, err := artifactStore.WriteDatasetSummaryMarkdownArtifact(datasetSummaryReport(profile))
	if err != nil {
		return fmt.Errorf("write smoke artifact: %w", err)
	}
	if _, err := artifactStore.ReadArtifactText(artifact.RelPath); err != nil {
		return fmt.Errorf("read smoke artifact: %w", err)
	}
	addCheck("artifact-write-read", artifact.RelPath)

	issueReport, err := issuereportSvc.Export(issuereportSvc.Options{
		WorkspaceRoot:     absRoot,
		OutputDir:         filepath.Join(absRoot, ".nexusdesk", "smoke", "issue-reports"),
		DiagnosticsReport: "Smoke diagnostics for platform build. Authorization: Bearer smoke-secret",
		ActivityTail:      []string{"Smoke opened workspace", "Smoke exported redacted issue report"},
		Now:               time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("export smoke issue report: %w", err)
	}
	if len(issueReport.Files) == 0 {
		return errors.New("smoke issue report did not include files")
	}
	addCheck("diagnostics-export", filepath.ToSlash(issueReport.Path))

	if writer == nil {
		writer = io.Discard
	}
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeSmokeFixtures(root string) error {
	files := map[string]string{
		"smoke-source.md": "# NexusDesk Smoke\n\nThis file contains platform-smoke-token for packaged app validation.\n",
		"data/smoke.csv":  "name,value\nalpha,1\nbeta,2\n",
	}
	for relPath, content := range files {
		absPath := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func datasetSummaryReport(profile datasetsSvc.Profile) artifactsSvc.DatasetSummaryReport {
	columns := make([]artifactsSvc.DatasetSummaryColumnReport, 0, len(profile.Columns))
	for _, column := range profile.Columns {
		columns = append(columns, artifactsSvc.DatasetSummaryColumnReport{
			Name:     column.Name,
			Type:     column.Type,
			NonEmpty: column.NonEmpty,
			Empty:    column.Empty,
			Samples:  append([]string{}, column.Samples...),
		})
	}
	return artifactsSvc.DatasetSummaryReport{
		Title:      "Packaged App Smoke Dataset",
		SourcePath: profile.RelPath,
		Format:     profile.Format,
		MediaType:  profile.MediaType,
		Size:       profile.Size,
		Rows:       profile.Rows,
		Columns:    columns,
		Sheet:      profile.Sheet,
		Sheets:     append([]string{}, profile.Sheets...),
		Truncated:  profile.Truncated,
		Notes:      append([]string{}, profile.Notes...),
	}
}

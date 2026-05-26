package main

import (
	"errors"
	"time"

	"NexusAugenticStudio/internal/appmeta"
	"NexusAugenticStudio/internal/artifact"
	"NexusAugenticStudio/internal/workspace"
)

type ArtifactService struct {
	workspaceRoot           func() string
	mirrorMetadataStore     func(root string, create bool) (appmeta.SQLiteStatus, error)
	listArtifactsFromMirror func(root string) ([]artifact.WorkspaceArtifact, error)
	persistMetadata         func(root string, relPath string)
	recordApproval          func(action string, target string, risk string, message string) string
}

func NewArtifactService(
	workspaceRoot func() string,
	mirrorMetadataStore func(string, bool) (appmeta.SQLiteStatus, error),
	listArtifactsFromMirror func(string) ([]artifact.WorkspaceArtifact, error),
	persistMetadata func(string, string),
	recordApproval func(string, string, string, string) string,
) *ArtifactService {
	return &ArtifactService{
		workspaceRoot:           workspaceRoot,
		mirrorMetadataStore:     mirrorMetadataStore,
		listArtifactsFromMirror: listArtifactsFromMirror,
		persistMetadata:         persistMetadata,
		recordApproval:          recordApproval,
	}
}

func (s *ArtifactService) CreateMarkdownReport(relPath string) (artifact.MarkdownReport, error) {
	root, err := s.requireRoot("creating reports")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}

	source := workspace.FilePreview{
		RelPath: relPath,
		Name:    "workspace-report",
	}
	if relPath != "" {
		preview, err := workspace.Preview(root, relPath, workspace.PreviewOptions{MaxBytes: chatContextFallbackMaxBytes})
		if err != nil {
			return artifact.MarkdownReport{}, err
		}
		source = preview
	}

	report, err := artifact.CreateMarkdownReport(root, source, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	s.persist(root, report.RelPath)
	s.record("artifact.report", report.RelPath, "low", report.Message)
	return report, nil
}

func (s *ArtifactService) CreateScanReport() (artifact.MarkdownReport, error) {
	root, err := s.requireRoot("creating scan reports")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}

	snapshot, err := workspace.Scan(root, workspace.ScanOptions{})
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	report, err := artifact.CreateScanReportMarkdown(root, snapshot, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	s.persist(root, report.RelPath)
	s.record("artifact.scan-report", report.RelPath, "low", report.Message)
	return report, nil
}

func (s *ArtifactService) CreateGeneratedMarkdown(request artifact.MarkdownArtifactRequest) (artifact.MarkdownReport, error) {
	root, err := s.requireRoot("creating artifacts")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}

	report, err := artifact.CreateGeneratedMarkdown(root, request, time.Now())
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	s.persist(root, report.RelPath)
	s.record("artifact.markdown", report.RelPath, "low", report.Message)
	return report, nil
}

func (s *ArtifactService) List() ([]artifact.WorkspaceArtifact, error) {
	root := s.workspaceRoot()
	if root == "" {
		return []artifact.WorkspaceArtifact{}, nil
	}
	if appmeta.Exists(root) && s.mirrorMetadataStore != nil && s.listArtifactsFromMirror != nil {
		if _, err := s.mirrorMetadataStore(root, false); err == nil {
			if items, readErr := s.listArtifactsFromMirror(root); readErr == nil {
				return items, nil
			}
		}
	}
	return artifact.List(root)
}

func (s *ArtifactService) Metadata(relPath string) (artifact.ArtifactMetadata, error) {
	root, err := s.requireRoot("reading artifact metadata")
	if err != nil {
		return artifact.ArtifactMetadata{}, err
	}
	return artifact.Metadata(root, relPath)
}

func (s *ArtifactService) Archive(relPath string) (artifact.MarkdownReport, error) {
	root, err := s.requireRoot("archiving artifacts")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}

	report, err := artifact.Archive(root, relPath)
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	s.record("artifact.archive", relPath, "medium", report.Message)
	return report, nil
}

func (s *ArtifactService) Delete(relPath string) (artifact.MarkdownReport, error) {
	root, err := s.requireRoot("deleting artifacts")
	if err != nil {
		return artifact.MarkdownReport{}, err
	}

	report, err := artifact.Delete(root, relPath)
	if err != nil {
		return artifact.MarkdownReport{}, err
	}
	s.record("artifact.delete", relPath, "high", report.Message)
	return report, nil
}

func (s *ArtifactService) Compare(leftRelPath string, rightRelPath string) (artifact.ArtifactComparison, error) {
	root, err := s.requireRoot("comparing artifacts")
	if err != nil {
		return artifact.ArtifactComparison{}, err
	}
	return artifact.Compare(root, leftRelPath, rightRelPath)
}

func (s *ArtifactService) requireRoot(action string) (string, error) {
	root := s.workspaceRoot()
	if root == "" {
		return "", errors.New("open a workspace before " + action)
	}
	return root, nil
}

func (s *ArtifactService) persist(root string, relPath string) {
	if s.persistMetadata != nil {
		s.persistMetadata(root, relPath)
	}
}

func (s *ArtifactService) record(action string, target string, risk string, message string) {
	if s.recordApproval != nil {
		s.recordApproval(action, target, risk, message)
	}
}

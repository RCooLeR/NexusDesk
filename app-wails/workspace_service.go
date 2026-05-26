package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"NexusAugenticStudio/internal/artifact"
	"NexusAugenticStudio/internal/storage"
	"NexusAugenticStudio/internal/workspace"
)

type WorkspaceService struct {
	chatStore      *storage.ChatHistoryStore
	recentStore    *storage.RecentWorkspaceStore
	recordApproval func(action string, target string, risk string, message string) string

	rootMu        sync.RWMutex
	workspaceRoot string
	watchMu       sync.Mutex
	fingerprints  map[string]workspace.FileFingerprint
}

func NewWorkspaceService(recentStore *storage.RecentWorkspaceStore, chatStore *storage.ChatHistoryStore, recordApproval func(string, string, string, string) string) *WorkspaceService {
	return &WorkspaceService{
		chatStore:      chatStore,
		recentStore:    recentStore,
		recordApproval: recordApproval,
	}
}

func (s *WorkspaceService) Open(root string) (WorkspaceOpenResult, error) {
	info, err := os.Stat(root)
	if err != nil {
		return WorkspaceOpenResult{}, err
	}
	if !info.IsDir() {
		return WorkspaceOpenResult{}, errors.New("workspace root must be a directory")
	}

	snapshot, err := workspace.Scan(root, workspace.ScanOptions{})
	if err != nil {
		return WorkspaceOpenResult{}, err
	}

	s.SetRoot(snapshot.Root)
	s.ResetFreshness(snapshot.Root)
	if _, err := s.recentStore.Add(snapshot.Root); err != nil {
		return WorkspaceOpenResult{}, err
	}

	return WorkspaceOpenResult{
		Selected: true,
		Snapshot: snapshot,
	}, nil
}

func (s *WorkspaceService) Refresh() (WorkspaceOpenResult, error) {
	root := s.Root()
	if root == "" {
		return WorkspaceOpenResult{Selected: false}, nil
	}

	snapshot, err := workspace.Scan(root, workspace.ScanOptions{})
	if err != nil {
		return WorkspaceOpenResult{}, err
	}

	return WorkspaceOpenResult{
		Selected: true,
		Snapshot: snapshot,
	}, nil
}

func (s *WorkspaceService) Search(query string) ([]workspace.SearchResult, error) {
	return s.search(query, workspace.SearchOptions{MaxResults: 70})
}

func (s *WorkspaceService) SearchAdvanced(request WorkspaceSearchRequest) ([]workspace.SearchResult, error) {
	return s.search(request.Query, workspace.SearchOptions{MaxResults: 70, Regex: request.Regex, Symbols: request.Symbols})
}

func (s *WorkspaceService) search(query string, options workspace.SearchOptions) ([]workspace.SearchResult, error) {
	root := s.Root()
	if root == "" {
		return []workspace.SearchResult{}, errors.New("open a workspace before searching")
	}

	results, err := workspace.Search(root, query, options)
	if err != nil {
		return nil, err
	}

	artifactResults, err := artifact.Search(root, query)
	if err != nil {
		return nil, err
	}
	results = append(results, artifactResults...)

	chatMessages, err := s.chatStore.Search(root, query)
	if err != nil {
		return nil, err
	}
	for _, message := range chatMessages {
		results = append(results, workspace.SearchResult{
			RelPath:   "Chat history",
			Name:      "Chat history",
			Kind:      "chat",
			FileType:  "chat",
			MatchType: message.Role,
			Snippet:   trimAppSnippet(message.Content),
		})
	}
	if len(results) > 100 {
		results = results[:100]
	}
	return results, nil
}

func (s *WorkspaceService) ReadFile(relPath string) (workspace.FilePreview, error) {
	root, err := s.requireRoot("reading files")
	if err != nil {
		return workspace.FilePreview{}, err
	}
	return workspace.Preview(root, relPath, workspace.PreviewOptions{})
}

func (s *WorkspaceService) Problems() (workspace.ProblemSummary, error) {
	root, err := s.requireRoot("scanning workspace problems")
	if err != nil {
		return workspace.ProblemSummary{}, err
	}
	return workspace.ScanProblems(root, 80)
}

func (s *WorkspaceService) PreviewFileWrite(request workspace.FileWriteRequest) (workspace.FileWriteProposal, error) {
	root, err := s.requireRoot("previewing file writes")
	if err != nil {
		return workspace.FileWriteProposal{}, err
	}
	return workspace.PreviewFileWrite(root, request)
}

func (s *WorkspaceService) ApplyFileWrite(request workspace.FileWriteRequest) (workspace.FileWriteProposal, error) {
	root, err := s.requireRoot("applying file writes")
	if err != nil {
		return workspace.FileWriteProposal{}, err
	}
	rollback, err := workspace.PrepareRollback(root, "file.write", request.RelPath, []string{request.RelPath})
	if err != nil {
		return workspace.FileWriteProposal{}, err
	}
	proposal, err := workspace.ApplyFileWrite(root, request)
	if err != nil {
		return workspace.FileWriteProposal{}, err
	}
	if _, err := workspace.CommitRollback(root, rollback); err != nil {
		return workspace.FileWriteProposal{}, err
	}
	s.record("file.write", proposal.RelPath, "medium", proposal.Message)
	return proposal, nil
}

func (s *WorkspaceService) PreviewFileDelete(relPath string) (workspace.FileDeleteProposal, error) {
	root, err := s.requireRoot("previewing file deletes")
	if err != nil {
		return workspace.FileDeleteProposal{}, err
	}
	return workspace.PreviewFileDelete(root, relPath)
}

func (s *WorkspaceService) ApplyFileDelete(relPath string) (workspace.FileDeleteProposal, error) {
	root, err := s.requireRoot("deleting files")
	if err != nil {
		return workspace.FileDeleteProposal{}, err
	}
	rollback, err := workspace.PrepareRollback(root, "file.delete", relPath, []string{relPath})
	if err != nil {
		return workspace.FileDeleteProposal{}, err
	}
	proposal, err := workspace.ApplyFileDelete(root, relPath)
	if err != nil {
		return workspace.FileDeleteProposal{}, err
	}
	if _, err := workspace.CommitRollback(root, rollback); err != nil {
		return workspace.FileDeleteProposal{}, err
	}
	s.record("file.delete", proposal.RelPath, "high", proposal.Message)
	return proposal, nil
}

func (s *WorkspaceService) PreviewFileMove(request workspace.FileMoveRequest) (workspace.FileMoveProposal, error) {
	root, err := s.requireRoot("previewing file moves")
	if err != nil {
		return workspace.FileMoveProposal{}, err
	}
	return workspace.PreviewFileMove(root, request)
}

func (s *WorkspaceService) ApplyFileMove(request workspace.FileMoveRequest) (workspace.FileMoveProposal, error) {
	root, err := s.requireRoot("moving files")
	if err != nil {
		return workspace.FileMoveProposal{}, err
	}
	rollback, err := workspace.PrepareRollback(root, "file.move", request.TargetRelPath, []string{request.SourceRelPath, request.TargetRelPath})
	if err != nil {
		return workspace.FileMoveProposal{}, err
	}
	proposal, err := workspace.ApplyFileMove(root, request)
	if err != nil {
		return workspace.FileMoveProposal{}, err
	}
	if _, err := workspace.CommitRollback(root, rollback); err != nil {
		return workspace.FileMoveProposal{}, err
	}
	s.record("file.move", proposal.TargetRelPath, "high", proposal.Message)
	return proposal, nil
}

func (s *WorkspaceService) PreviewFileCopy(request workspace.FileCopyRequest) (workspace.FileCopyProposal, error) {
	root, err := s.requireRoot("previewing file copies")
	if err != nil {
		return workspace.FileCopyProposal{}, err
	}
	return workspace.PreviewFileCopy(root, request)
}

func (s *WorkspaceService) ApplyFileCopy(request workspace.FileCopyRequest) (workspace.FileCopyProposal, error) {
	root, err := s.requireRoot("copying files")
	if err != nil {
		return workspace.FileCopyProposal{}, err
	}
	rollback, err := workspace.PrepareRollback(root, "file.copy", request.TargetRelPath, []string{request.TargetRelPath})
	if err != nil {
		return workspace.FileCopyProposal{}, err
	}
	proposal, err := workspace.ApplyFileCopy(root, request)
	if err != nil {
		return workspace.FileCopyProposal{}, err
	}
	if _, err := workspace.CommitRollback(root, rollback); err != nil {
		return workspace.FileCopyProposal{}, err
	}
	s.record("file.copy", proposal.TargetRelPath, "medium", proposal.Message)
	return proposal, nil
}

func (s *WorkspaceService) ListRollbacks() ([]workspace.RollbackRecord, error) {
	root, err := s.requireRoot("listing rollbacks")
	if err != nil {
		return []workspace.RollbackRecord{}, err
	}
	return workspace.ListRollbacks(root)
}

func (s *WorkspaceService) ApplyRollback(id string) (workspace.RollbackApplyResult, error) {
	root, err := s.requireRoot("applying rollback")
	if err != nil {
		return workspace.RollbackApplyResult{}, err
	}
	result, err := workspace.ApplyRollback(root, id)
	if err != nil {
		return workspace.RollbackApplyResult{}, err
	}
	s.record("file.rollback", result.ID, "high", result.Message)
	return result, nil
}

func (s *WorkspaceService) CheckFreshness() (workspace.FreshnessStatus, error) {
	root, err := s.requireRoot("checking file changes")
	if err != nil {
		return workspace.FreshnessStatus{}, err
	}
	current, err := workspace.SnapshotFingerprints(root)
	if err != nil {
		return workspace.FreshnessStatus{}, err
	}
	s.watchMu.Lock()
	previous := s.fingerprints
	s.fingerprints = current
	s.watchMu.Unlock()
	if previous == nil {
		return workspace.FreshnessStatus{
			Changed:        []workspace.FileChange{},
			StaleArtifacts: []string{},
			StaleDatasets:  []string{},
			Message:        "Workspace watcher baseline captured.",
		}, nil
	}

	changes := workspace.CompareFingerprints(previous, current)
	staleArtifacts := s.StaleArtifactsForChanges(root, changes)
	staleDatasets := staleDatasetsForChanges(changes)
	message := "Workspace files are current."
	if len(changes) > 0 {
		message = fmt.Sprintf("%d workspace file changes detected.", len(changes))
	}
	if len(staleArtifacts) > 0 {
		message = fmt.Sprintf("%s %d artifacts may be stale.", message, len(staleArtifacts))
	}
	if len(staleDatasets) > 0 {
		message = fmt.Sprintf("%s %d dataset-derived views need refresh.", message, len(staleDatasets))
	}
	return workspace.FreshnessStatus{
		Changed:        changes,
		StaleArtifacts: staleArtifacts,
		StaleDatasets:  staleDatasets,
		Message:        message,
	}, nil
}

func (s *WorkspaceService) Root() string {
	s.rootMu.RLock()
	defer s.rootMu.RUnlock()
	return s.workspaceRoot
}

func (s *WorkspaceService) SetRoot(root string) {
	s.rootMu.Lock()
	defer s.rootMu.Unlock()
	s.workspaceRoot = root
}

func (s *WorkspaceService) ResetFreshness(root string) {
	fingerprints, err := workspace.SnapshotFingerprints(root)
	if err != nil {
		return
	}
	s.watchMu.Lock()
	s.fingerprints = fingerprints
	s.watchMu.Unlock()
}

func (s *WorkspaceService) StaleArtifactsForChanges(root string, changes []workspace.FileChange) []string {
	if len(changes) == 0 {
		return nil
	}
	changed := map[string]bool{}
	for _, change := range changes {
		changed[filepath.ToSlash(change.RelPath)] = true
	}
	items, err := artifact.List(root)
	if err != nil {
		return nil
	}
	stale := []string{}
	for _, item := range items {
		metadata, err := artifact.Metadata(root, item.RelPath)
		if err != nil {
			continue
		}
		sourcePaths := append([]string{}, metadata.SourcePaths...)
		if metadata.ContextRelPath != "" {
			sourcePaths = append(sourcePaths, metadata.ContextRelPath)
		}
		for _, source := range sourcePaths {
			if changed[filepath.ToSlash(source)] {
				stale = append(stale, item.RelPath)
				break
			}
		}
	}
	return stale
}

func (s *WorkspaceService) requireRoot(action string) (string, error) {
	root := s.Root()
	if root == "" {
		return "", fmt.Errorf("open a workspace before %s", action)
	}
	return root, nil
}

func (s *WorkspaceService) record(action string, target string, risk string, message string) {
	if s.recordApproval != nil {
		s.recordApproval(action, target, risk, message)
	}
}

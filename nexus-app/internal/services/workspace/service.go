package workspace

import (
	"path/filepath"

	"nexusdesk/internal/domain"
)

const (
	defaultEntryLimit      = 600
	defaultPreviewByteSize = 256 * 1024
)

type Service struct {
	entryLimit       int
	previewByteLimit int64
}

func New() *Service {
	return &Service{
		entryLimit:       defaultEntryLimit,
		previewByteLimit: defaultPreviewByteSize,
	}
}

func (s *Service) Open(root string) (domain.Workspace, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return domain.Workspace{}, err
	}
	listing, err := s.ListChildren(absRoot, "")
	if err != nil {
		return domain.Workspace{}, err
	}
	return domain.Workspace{
		Root:    absRoot,
		Name:    filepath.Base(absRoot),
		Summary: listing.Summary,
		Tree:    listing.Nodes,
	}, nil
}

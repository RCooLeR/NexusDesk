package artifacts

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const artifactsDirRelPath = ".nexusdesk/artifacts"

type Store struct {
	root string
}

func NewStore(root string) (*Store, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("artifact root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("artifact root must be a directory")
	}
	return &Store{root: absRoot}, nil
}

func (s *Store) relPath(parts ...string) string {
	all := append([]string{artifactsDirRelPath}, parts...)
	return filepath.ToSlash(filepath.Join(all...))
}

func (s *Store) absPath(relPath string) string {
	return filepath.Join(s.root, filepath.FromSlash(relPath))
}

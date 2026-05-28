package artifacts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func (s *Store) createUniqueArtifactFile(subdir string, title string, extension string, createdAt time.Time) (string, string, *os.File, error) {
	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}
	baseName := fmt.Sprintf("%s-%s", artifactTimestamp(createdAt), safeName(title))
	for attempt := 0; attempt < 100; attempt++ {
		name := baseName
		if attempt > 0 {
			name = fmt.Sprintf("%s-%02d", baseName, attempt+1)
		}
		relPath := s.relPath(subdir, name+extension)
		absPath := s.absPath(relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return "", "", nil, err
		}
		file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return relPath, absPath, file, err
	}
	return "", "", nil, fmt.Errorf("could not create a unique artifact file for %q", title)
}

func artifactTimestamp(value time.Time) string {
	value = value.UTC()
	return fmt.Sprintf("%s-%09d", value.Format("20060102-150405"), value.Nanosecond())
}

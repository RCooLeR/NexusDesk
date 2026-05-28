package operations

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultFileCap     = 800
	defaultReadMaxByte = 256 * 1024
)

type Service struct {
	fileCap     int
	readMaxByte int64
}

func New() *Service {
	return &Service{fileCap: defaultFileCap, readMaxByte: defaultReadMaxByte}
}

func (s *Service) Scan(root string) (ScanResult, error) {
	return s.ScanContext(context.Background(), root)
}

func (s *Service) ScanContext(ctx context.Context, root string) (ScanResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ScanResult{}, ctx.Err()
	default:
	}
	absRoot, err := cleanRoot(root)
	if err != nil {
		return ScanResult{}, err
	}
	result := ScanResult{Files: []File{}}
	err = filepath.WalkDir(absRoot, func(path string, entry os.DirEntry, walkErr error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if walkErr != nil {
			result.Summary.Unreadable++
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == absRoot {
			return nil
		}
		relPath := relFile(absRoot, path)
		if entry.IsDir() {
			if shouldSkipDir(entry.Name(), relPath) {
				result.Summary.SkippedDirs++
				return filepath.SkipDir
			}
			return nil
		}
		if len(result.Files) >= s.fileCap {
			result.Summary.EntryCap++
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			result.Summary.Unreadable++
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		file, ok := classifyFile(relPath, info.Size())
		if !ok {
			return nil
		}
		result.Files = append(result.Files, file)
		countKind(&result.Summary, file.Kind)
		return nil
	})
	if err != nil {
		return ScanResult{}, err
	}
	sort.Slice(result.Files, func(left, right int) bool {
		if result.Files[left].Kind != result.Files[right].Kind {
			return result.Files[left].Kind < result.Files[right].Kind
		}
		return strings.ToLower(result.Files[left].RelPath) < strings.ToLower(result.Files[right].RelPath)
	})
	result.Summary.Files = len(result.Files)
	result.Message = fmt.Sprintf("%d operations files found: %d Compose, %d Dockerfiles, %d env, %d config, %d logs, %d scripts.",
		result.Summary.Files,
		result.Summary.Compose,
		result.Summary.Dockerfiles,
		result.Summary.Env,
		result.Summary.Config,
		result.Summary.Logs,
		result.Summary.Scripts,
	)
	return result, nil
}

func (s *Service) Inspect(root string, relPath string) (Inspection, error) {
	return s.InspectContext(context.Background(), root, relPath)
}

func (s *Service) InspectContext(ctx context.Context, root string, relPath string) (Inspection, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return Inspection{}, ctx.Err()
	default:
	}
	_, cleanRelPath, target, info, err := resolveFile(root, relPath)
	if err != nil {
		return Inspection{}, err
	}
	file, ok := classifyFile(cleanRelPath, info.Size())
	if !ok {
		return Inspection{}, errors.New("selected file is not an operations context file")
	}
	readLimit := s.readMaxByte
	truncated := false
	if info.Size() > readLimit {
		truncated = true
	}
	select {
	case <-ctx.Done():
		return Inspection{}, ctx.Err()
	default:
	}
	content, err := readBounded(target, readLimit)
	if err != nil {
		return Inspection{}, err
	}
	select {
	case <-ctx.Done():
		return Inspection{}, ctx.Err()
	default:
	}
	text := string(content)
	warnings := []string{"Read-only inspection only. Nexus did not run Docker, shell, or service commands."}
	if truncated {
		warnings = append(warnings, fmt.Sprintf("File was truncated to %d bytes for interactive inspection.", readLimit))
	}
	if strings.ContainsRune(text, '\x00') {
		return Inspection{}, errors.New("operations inspector only supports text-like files")
	}
	inspection := Inspection{
		File:      file,
		Text:      redactSecrets(text),
		Truncated: truncated,
		Warnings:  warnings,
	}
	if file.Kind == FileKindCompose {
		inspection.Services = ParseComposeServices(inspection.Text)
		inspection.Topology = BuildComposeTopology(inspection.Services)
	}
	return inspection, nil
}

func countKind(summary *Summary, kind FileKind) {
	switch kind {
	case FileKindDockerfile:
		summary.Dockerfiles++
	case FileKindCompose:
		summary.Compose++
	case FileKindEnv:
		summary.Env++
	case FileKindConfig:
		summary.Config++
	case FileKindLog:
		summary.Logs++
	case FileKindScript:
		summary.Scripts++
	}
}

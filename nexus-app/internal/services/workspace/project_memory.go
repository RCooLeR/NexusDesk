package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	projectMemoryDirRelPath    = ".nexusdesk/project-memory"
	projectMemoryFileName      = "memory.json"
	projectMemoryVersion       = 1
	projectMemoryMaxRecords    = 200
	projectMemoryMaxContentLen = 16 * 1024
	projectMemoryMaxSources    = 20
)

var projectMemoryKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,95}$`)

type ProjectMemoryUpdateRequest struct {
	Key            string
	Content        string
	SourceRelPaths []string
}

type ProjectMemoryRecord struct {
	Key            string    `json:"key"`
	Content        string    `json:"content"`
	SourceRelPaths []string  `json:"sourceRelPaths,omitempty"`
	SourceSHA256   string    `json:"sourceSha256,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type ProjectMemoryUpdateResult struct {
	Record  ProjectMemoryRecord
	Created bool
	Count   int
	Message string
}

type projectMemoryFile struct {
	Version int                   `json:"version"`
	Records []ProjectMemoryRecord `json:"records"`
}

func (s *Service) UpdateProjectMemory(root string, request ProjectMemoryUpdateRequest) (ProjectMemoryUpdateResult, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return ProjectMemoryUpdateResult{}, err
	}
	key := strings.TrimSpace(request.Key)
	if !projectMemoryKeyPattern.MatchString(key) {
		return ProjectMemoryUpdateResult{}, errors.New("project memory key must start with a letter or number and contain only letters, numbers, _, ., :, or -")
	}
	content := strings.TrimSpace(request.Content)
	if content == "" {
		return ProjectMemoryUpdateResult{}, errors.New("project memory content is required")
	}
	if len(content) > projectMemoryMaxContentLen {
		return ProjectMemoryUpdateResult{}, errors.New("project memory content is too large")
	}
	sources, fingerprint, err := validateProjectMemorySources(absRoot, request.SourceRelPaths)
	if err != nil {
		return ProjectMemoryUpdateResult{}, err
	}
	memory, err := readProjectMemoryFile(absRoot)
	if err != nil {
		return ProjectMemoryUpdateResult{}, err
	}
	now := time.Now().UTC()
	created := true
	record := ProjectMemoryRecord{
		Key:            key,
		Content:        content,
		SourceRelPaths: sources,
		SourceSHA256:   fingerprint,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	for index := range memory.Records {
		if memory.Records[index].Key != key {
			continue
		}
		created = false
		record.CreatedAt = memory.Records[index].CreatedAt
		if record.CreatedAt.IsZero() {
			record.CreatedAt = now
		}
		memory.Records[index] = record
		if err := writeProjectMemoryFile(absRoot, memory); err != nil {
			return ProjectMemoryUpdateResult{}, err
		}
		return ProjectMemoryUpdateResult{
			Record:  record,
			Created: false,
			Count:   len(memory.Records),
			Message: fmt.Sprintf("Updated project memory %q with %d source(s).", key, len(sources)),
		}, nil
	}
	if len(memory.Records) >= projectMemoryMaxRecords {
		return ProjectMemoryUpdateResult{}, errors.New("project memory record limit reached")
	}
	memory.Records = append(memory.Records, record)
	sort.SliceStable(memory.Records, func(left int, right int) bool {
		return strings.ToLower(memory.Records[left].Key) < strings.ToLower(memory.Records[right].Key)
	})
	if err := writeProjectMemoryFile(absRoot, memory); err != nil {
		return ProjectMemoryUpdateResult{}, err
	}
	return ProjectMemoryUpdateResult{
		Record:  record,
		Created: created,
		Count:   len(memory.Records),
		Message: fmt.Sprintf("Created project memory %q with %d source(s).", key, len(sources)),
	}, nil
}

func validateProjectMemorySources(absRoot string, values []string) ([]string, string, error) {
	if len(values) > projectMemoryMaxSources {
		return nil, "", errors.New("too many project memory sources")
	}
	sources := []string{}
	seen := map[string]bool{}
	hash := sha256.New()
	for _, value := range values {
		cleanRelPath, err := cleanRel(value)
		if err != nil {
			return nil, "", err
		}
		if cleanRelPath == "" {
			continue
		}
		if isInternalMetadataPath(cleanRelPath) {
			return nil, "", errors.New("project memory sources cannot point at Nexus metadata")
		}
		absTarget, _, _, err := resolveFile(absRoot, cleanRelPath)
		if err != nil {
			return nil, "", err
		}
		cleanRelPath = filepath.ToSlash(cleanRelPath)
		if seen[cleanRelPath] {
			continue
		}
		seen[cleanRelPath] = true
		sources = append(sources, cleanRelPath)
		content, err := readFilePrefix(absTarget, 64*1024)
		if err != nil {
			return nil, "", err
		}
		hash.Write([]byte(cleanRelPath))
		hash.Write([]byte{0})
		hash.Write(content)
	}
	if len(sources) == 0 {
		return sources, "", nil
	}
	return sources, hex.EncodeToString(hash.Sum(nil)), nil
}

func readProjectMemoryFile(absRoot string) (projectMemoryFile, error) {
	path := filepath.Join(absRoot, filepath.FromSlash(projectMemoryDirRelPath), projectMemoryFileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return projectMemoryFile{Version: projectMemoryVersion, Records: []ProjectMemoryRecord{}}, nil
	}
	if err != nil {
		return projectMemoryFile{}, err
	}
	var memory projectMemoryFile
	if err := json.Unmarshal(data, &memory); err != nil {
		return projectMemoryFile{}, err
	}
	if memory.Version == 0 {
		memory.Version = projectMemoryVersion
	}
	if memory.Records == nil {
		memory.Records = []ProjectMemoryRecord{}
	}
	return memory, nil
}

func writeProjectMemoryFile(absRoot string, memory projectMemoryFile) error {
	memory.Version = projectMemoryVersion
	dir := filepath.Join(absRoot, filepath.FromSlash(projectMemoryDirRelPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(memory, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tempPath := filepath.Join(dir, "."+projectMemoryFileName+".tmp")
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return err
	}
	targetPath := filepath.Join(dir, projectMemoryFileName)
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tempPath, targetPath)
}

package git

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (s *Service) History(root string, relPath string, limit int) (HistoryResult, error) {
	generatedAt := time.Now().UTC()
	result := HistoryResult{
		Path:        cleanOptionalRelPath(relPath),
		Limit:       boundedHistoryLimit(limit),
		Entries:     []HistoryEntry{},
		GeneratedAt: generatedAt,
	}
	root = strings.TrimSpace(root)
	if root == "" {
		result.Message = "Open a workspace before reading Git history."
		return result, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return HistoryResult{}, err
	}
	if _, err := gitOutputFor(absRoot, operationStatus, "rev-parse", "--show-toplevel"); err != nil {
		result.Message = repositoryUnavailableMessage(absRoot, err)
		return result, nil
	}
	args := []string{
		"log",
		fmt.Sprintf("--max-count=%d", result.Limit+1),
		"--date=iso-strict",
		"--pretty=format:%H%x1f%h%x1f%an%x1f%ae%x1f%ad%x1f%s",
	}
	if result.Path != "" {
		cleanPath, err := cleanRelPath(result.Path)
		if err != nil {
			result.Message = err.Error()
			return result, nil
		}
		result.Path = cleanPath
		args = append(args, "--", cleanPath)
	}
	output, err := gitOutputFor(absRoot, operationHistory, args...)
	if err != nil {
		result.Message = "Could not read Git history: " + err.Error()
		return result, nil
	}
	result.Available = true
	result.Entries, result.Truncated = parseHistory(output, result.Limit)
	result.Message = fmt.Sprintf("Loaded %d Git history entries.", len(result.Entries))
	if len(result.Entries) == 1 {
		result.Message = "Loaded 1 Git history entry."
	}
	if result.Path != "" {
		result.Message = fmt.Sprintf("Loaded %d Git history entries for %s.", len(result.Entries), result.Path)
		if len(result.Entries) == 1 {
			result.Message = "Loaded 1 Git history entry for " + result.Path + "."
		}
	}
	return result, nil
}

func (s *Service) Blame(root string, relPath string, startLine int, endLine int) (BlameResult, error) {
	generatedAt := time.Now().UTC()
	result := BlameResult{
		Path:        cleanOptionalRelPath(relPath),
		Lines:       []BlameLine{},
		GeneratedAt: generatedAt,
	}
	root = strings.TrimSpace(root)
	if root == "" {
		result.Message = "Open a workspace before reading Git blame."
		return result, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return BlameResult{}, err
	}
	cleanPath, err := cleanRelPath(result.Path)
	if err != nil {
		result.Message = err.Error()
		return result, nil
	}
	result.Path = cleanPath
	if _, err := gitOutputFor(absRoot, operationStatus, "rev-parse", "--show-toplevel"); err != nil {
		result.Message = repositoryUnavailableMessage(absRoot, err)
		return result, nil
	}
	startLine, endLine = boundedBlameRange(startLine, endLine)
	result.StartLine = startLine
	result.EndLine = endLine
	args := []string{"blame", "--line-porcelain", "-L", fmt.Sprintf("%d,%d", startLine, endLine), "--", cleanPath}
	output, err := gitOutputFor(absRoot, operationHistory, args...)
	if err != nil {
		result.Message = "Could not read Git blame: " + err.Error()
		return result, nil
	}
	result.Available = true
	result.Lines, result.Truncated = parseBlame(output, blameMaxLines)
	result.Message = fmt.Sprintf("Loaded %d Git blame lines for %s.", len(result.Lines), cleanPath)
	if len(result.Lines) == 1 {
		result.Message = "Loaded 1 Git blame line for " + cleanPath + "."
	}
	return result, nil
}

func boundedHistoryLimit(limit int) int {
	if limit <= 0 {
		return DefaultHistoryLimit
	}
	if limit > historyMaxLimit {
		return historyMaxLimit
	}
	return limit
}

func boundedBlameRange(startLine int, endLine int) (int, int) {
	if startLine <= 0 {
		return 1, blameMaxLines
	}
	if endLine < startLine {
		endLine = startLine
	}
	if endLine-startLine+1 > blameMaxLines {
		endLine = startLine + blameMaxLines - 1
	}
	return startLine, endLine
}

func parseHistory(output string, limit int) ([]HistoryEntry, bool) {
	entries := []HistoryEntry{}
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\x1f", 6)
		if len(parts) != 6 {
			continue
		}
		if len(entries) >= limit {
			return entries, true
		}
		entries = append(entries, HistoryEntry{
			Hash:      strings.TrimSpace(parts[0]),
			ShortHash: strings.TrimSpace(parts[1]),
			Author:    strings.TrimSpace(parts[2]),
			Email:     strings.TrimSpace(parts[3]),
			Date:      strings.TrimSpace(parts[4]),
			Subject:   strings.TrimSpace(parts[5]),
		})
	}
	return entries, false
}

func parseBlame(output string, maxLines int) ([]BlameLine, bool) {
	lines := []BlameLine{}
	current := BlameLine{}
	for _, rawLine := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(rawLine) == "" && current.Hash == "" {
			continue
		}
		if strings.HasPrefix(rawLine, "\t") {
			if len(lines) >= maxLines {
				return lines, true
			}
			current.Content = strings.TrimPrefix(rawLine, "\t")
			current.ShortHash = shortHash(current.Hash)
			lines = append(lines, current)
			current = BlameLine{}
			continue
		}
		parts := strings.SplitN(rawLine, " ", 2)
		key := parts[0]
		value := ""
		if len(parts) == 2 {
			value = strings.TrimSpace(parts[1])
		}
		switch key {
		case "author":
			current.Author = value
		case "author-time":
			if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
				current.Date = time.Unix(seconds, 0).UTC().Format(time.RFC3339)
			}
		case "summary":
			current.Summary = value
		default:
			header := strings.Fields(rawLine)
			if len(header) >= 3 && len(header[0]) >= 7 {
				current.Hash = header[0]
				if finalLine, err := strconv.Atoi(header[2]); err == nil {
					current.Line = finalLine
				}
			}
		}
	}
	return lines, false
}

func shortHash(hash string) string {
	hash = strings.TrimSpace(hash)
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}

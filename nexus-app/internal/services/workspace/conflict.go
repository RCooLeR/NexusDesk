package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ConflictResolutionOurs   = "ours"
	ConflictResolutionTheirs = "theirs"
	ConflictResolutionBoth   = "both"
)

type ConflictResolutionResult struct {
	RelPath       string
	Strategy      string
	ConflictCount int
	Content       string
	Encoding      string
	Message       string
}

func (s *Service) ResolveConflictMarkers(root string, relPath string, strategy string) (ConflictResolutionResult, error) {
	_, absTarget, cleanRelPath, err := resolveWriteTarget(root, relPath)
	if err != nil {
		return ConflictResolutionResult{}, err
	}
	info, err := os.Lstat(absTarget)
	if os.IsNotExist(err) {
		return ConflictResolutionResult{}, errors.New("conflict resolution target does not exist")
	}
	if err != nil {
		return ConflictResolutionResult{}, err
	}
	if info.IsDir() {
		return ConflictResolutionResult{}, errors.New("conflict resolution target must be a file")
	}
	if info.Size() > writeContentMaxBytes {
		return ConflictResolutionResult{}, errors.New("conflict resolution target is too large")
	}
	content, err := os.ReadFile(absTarget)
	if err != nil {
		return ConflictResolutionResult{}, err
	}
	text, encoding, err := decodeText(content)
	if err != nil {
		return ConflictResolutionResult{}, err
	}
	if !appendDecodedTextSafe(text) {
		return ConflictResolutionResult{}, errors.New("conflict resolution target is not safe text")
	}
	normalizedStrategy, err := normalizeConflictResolutionStrategy(strategy)
	if err != nil {
		return ConflictResolutionResult{}, err
	}
	resolved, conflictCount, err := resolveConflictText(text, normalizedStrategy)
	if err != nil {
		return ConflictResolutionResult{}, err
	}
	resolvedRelPath := filepath.ToSlash(cleanRelPath)
	return ConflictResolutionResult{
		RelPath:       resolvedRelPath,
		Strategy:      normalizedStrategy,
		ConflictCount: conflictCount,
		Content:       resolved,
		Encoding:      encoding,
		Message:       fmt.Sprintf("Resolved %d conflict marker set(s) in %s using %s.", conflictCount, resolvedRelPath, normalizedStrategy),
	}, nil
}

func normalizeConflictResolutionStrategy(strategy string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case ConflictResolutionOurs, "our", "current", "local", "mine", "head":
		return ConflictResolutionOurs, nil
	case ConflictResolutionTheirs, "their", "incoming", "remote":
		return ConflictResolutionTheirs, nil
	case ConflictResolutionBoth, "all", "combine", "combined":
		return ConflictResolutionBoth, nil
	default:
		return "", errors.New("strategy must be one of: ours, theirs, both")
	}
}

func resolveConflictText(content string, strategy string) (string, int, error) {
	lineEnding := "\n"
	if strings.Contains(content, "\r\n") {
		lineEnding = "\r\n"
	}
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	hadFinalNewline := strings.HasSuffix(normalized, "\n")
	lines := strings.Split(normalized, "\n")
	if hadFinalNewline {
		lines = lines[:len(lines)-1]
	}

	const (
		stateNormal = iota
		stateOurs
		stateBase
		stateTheirs
	)
	state := stateNormal
	conflictCount := 0
	output := make([]string, 0, len(lines))
	ours := []string{}
	theirs := []string{}

	for index, line := range lines {
		switch state {
		case stateNormal:
			switch {
			case isConflictStart(line):
				conflictCount++
				ours = ours[:0]
				theirs = theirs[:0]
				state = stateOurs
			case isConflictSeparator(line), isConflictEnd(line), isConflictBase(line):
				return "", 0, fmt.Errorf("unexpected conflict marker at line %d", index+1)
			default:
				output = append(output, line)
			}
		case stateOurs:
			switch {
			case isConflictStart(line):
				return "", 0, fmt.Errorf("nested conflict marker at line %d", index+1)
			case isConflictBase(line):
				state = stateBase
			case isConflictSeparator(line):
				state = stateTheirs
			case isConflictEnd(line):
				return "", 0, fmt.Errorf("conflict marker set missing separator before line %d", index+1)
			default:
				ours = append(ours, line)
			}
		case stateBase:
			switch {
			case isConflictStart(line):
				return "", 0, fmt.Errorf("nested conflict marker at line %d", index+1)
			case isConflictSeparator(line):
				state = stateTheirs
			case isConflictEnd(line):
				return "", 0, fmt.Errorf("conflict marker set missing separator before line %d", index+1)
			default:
				// Diff3 base content is intentionally ignored for all supported strategies.
			}
		case stateTheirs:
			switch {
			case isConflictEnd(line):
				output = appendConflictResolution(output, ours, theirs, strategy)
				state = stateNormal
			case isConflictStart(line), isConflictSeparator(line), isConflictBase(line):
				return "", 0, fmt.Errorf("malformed conflict marker set near line %d", index+1)
			default:
				theirs = append(theirs, line)
			}
		}
	}
	if state != stateNormal {
		return "", 0, errors.New("unterminated conflict marker set")
	}
	if conflictCount == 0 {
		return "", 0, errors.New("no conflict markers were found")
	}
	resolved := strings.Join(output, "\n")
	if hadFinalNewline {
		resolved += "\n"
	}
	if lineEnding != "\n" {
		resolved = strings.ReplaceAll(resolved, "\n", lineEnding)
	}
	return resolved, conflictCount, nil
}

func appendConflictResolution(output []string, ours []string, theirs []string, strategy string) []string {
	switch strategy {
	case ConflictResolutionOurs:
		return append(output, ours...)
	case ConflictResolutionTheirs:
		return append(output, theirs...)
	case ConflictResolutionBoth:
		output = append(output, ours...)
		return append(output, theirs...)
	default:
		return output
	}
}

func isConflictStart(line string) bool {
	return strings.HasPrefix(line, "<<<<<<<")
}

func isConflictBase(line string) bool {
	return strings.HasPrefix(line, "|||||||")
}

func isConflictSeparator(line string) bool {
	return strings.HasPrefix(line, "=======")
}

func isConflictEnd(line string) bool {
	return strings.HasPrefix(line, ">>>>>>>")
}

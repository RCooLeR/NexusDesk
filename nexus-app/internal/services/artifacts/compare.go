package artifacts

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const artifactDiffMaxTotalLines = 10000

func (s *Store) CompareArtifacts(leftRelPath string, rightRelPath string) (ArtifactComparison, error) {
	left, err := s.artifactByPath(leftRelPath)
	if err != nil {
		return ArtifactComparison{}, err
	}
	right, err := s.artifactByPath(rightRelPath)
	if err != nil {
		return ArtifactComparison{}, err
	}
	if left.RelPath == right.RelPath {
		return ArtifactComparison{}, errors.New("artifact comparison requires two different artifacts")
	}
	if left.Kind != right.Kind {
		return ArtifactComparison{}, fmt.Errorf("artifact kinds must match: %s vs %s", left.Kind, right.Kind)
	}
	leftText, err := s.ReadArtifactText(left.RelPath)
	if err != nil {
		return ArtifactComparison{}, err
	}
	rightText, err := s.ReadArtifactText(right.RelPath)
	if err != nil {
		return ArtifactComparison{}, err
	}
	diff := buildArtifactDiff(left.RelPath, right.RelPath, leftText, rightText)
	same := leftText == rightText
	message := fmt.Sprintf("Compared %s with %s.", left.RelPath, right.RelPath)
	if same {
		message = "Artifacts have identical text content."
	}
	return ArtifactComparison{
		Kind:       left.Kind,
		LeftPath:   left.RelPath,
		RightPath:  right.RelPath,
		LeftTitle:  artifactTitle(left),
		RightTitle: artifactTitle(right),
		Diff:       diff,
		Same:       same,
		Message:    message,
	}, nil
}

func buildArtifactDiff(leftPath string, rightPath string, leftText string, rightText string) string {
	var builder strings.Builder
	builder.WriteString("--- ")
	builder.WriteString(leftPath)
	builder.WriteString("\n+++ ")
	builder.WriteString(rightPath)
	builder.WriteString("\n")
	leftLines := splitArtifactDiffLines(leftText)
	rightLines := splitArtifactDiffLines(rightText)
	if len(leftLines)+len(rightLines) > artifactDiffMaxTotalLines {
		builder.WriteString("Artifact comparison is too line-dense for inline diff.\n")
		builder.WriteString("Left lines: ")
		builder.WriteString(strconv.Itoa(len(leftLines)))
		builder.WriteString("\nRight lines: ")
		builder.WriteString(strconv.Itoa(len(rightLines)))
		builder.WriteString("\n")
		return builder.String()
	}
	for _, line := range lcsArtifactDiffLines(leftLines, rightLines) {
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	return builder.String()
}

func lcsArtifactDiffLines(leftLines []string, rightLines []string) []string {
	table := make([][]int, len(leftLines)+1)
	for index := range table {
		table[index] = make([]int, len(rightLines)+1)
	}
	for left := len(leftLines) - 1; left >= 0; left-- {
		for right := len(rightLines) - 1; right >= 0; right-- {
			if leftLines[left] == rightLines[right] {
				table[left][right] = table[left+1][right+1] + 1
			} else if table[left+1][right] >= table[left][right+1] {
				table[left][right] = table[left+1][right]
			} else {
				table[left][right] = table[left][right+1]
			}
		}
	}
	diff := []string{}
	left := 0
	right := 0
	for left < len(leftLines) && right < len(rightLines) {
		if leftLines[left] == rightLines[right] {
			diff = append(diff, " "+leftLines[left])
			left++
			right++
			continue
		}
		if table[left+1][right] >= table[left][right+1] {
			diff = append(diff, "-"+leftLines[left])
			left++
		} else {
			diff = append(diff, "+"+rightLines[right])
			right++
		}
	}
	for left < len(leftLines) {
		diff = append(diff, "-"+leftLines[left])
		left++
	}
	for right < len(rightLines) {
		diff = append(diff, "+"+rightLines[right])
		right++
	}
	return diff
}

func splitArtifactDiffLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return []string{}
	}
	return strings.Split(content, "\n")
}

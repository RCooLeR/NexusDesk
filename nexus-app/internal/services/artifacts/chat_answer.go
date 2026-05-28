package artifacts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const chatAnswerCitationSnippetLimit = 16
const chatAnswerCitationSnippetMaxRunes = 512

func (s *Store) WriteChatAnswer(report ChatAnswerReport) (Artifact, error) {
	content := strings.TrimSpace(report.Content)
	if content == "" {
		return Artifact{}, errors.New("assistant answer content is required")
	}
	createdAt := time.Now().UTC()
	title := strings.TrimSpace(report.Title)
	if title == "" {
		title = chatAnswerTitle(report.Prompt)
	}
	relPath := s.relPath("chat-answers", fmt.Sprintf("%s-%s.md", createdAt.Format("20060102-150405-000000000"), safeName(title)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	markdown := chatAnswerMarkdown(report, title, content, createdAt)
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	if _, err := file.WriteString(markdown); err != nil {
		return Artifact{}, err
	}
	metadata := Metadata{
		Kind:             "chat-answer",
		Title:            title,
		RelPath:          relPath,
		Source:           firstNonEmptyArtifact(report.Source, "Nexus assistant"),
		ContextRelPath:   strings.TrimSpace(report.ContextRelPath),
		Prompt:           strings.TrimSpace(report.Prompt),
		Model:            strings.TrimSpace(report.Model),
		SourcePaths:      append([]string{}, report.SourcePaths...),
		CitationRefs:     append([]string{}, report.CitationRefs...),
		CitationSnippets: cleanChatAnswerSnippets(report.CitationSnippets),
		EvidenceQuality:  strings.TrimSpace(report.EvidenceQuality),
		EvidenceSummary:  strings.TrimSpace(report.EvidenceSummary),
		GeneratedAt:      createdAt,
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:         metadata.Kind,
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      "Assistant response artifact created at " + relPath + ".",
		Size:         int64(len(markdown)),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  append([]string{}, report.SourcePaths...),
	}, nil
}

func chatAnswerMarkdown(report ChatAnswerReport, title string, content string, createdAt time.Time) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	writeKV(&builder, "Generated", formatArtifactTime(createdAt))
	writeKV(&builder, "Source", firstNonEmptyArtifact(report.Source, "Nexus assistant"))
	writeKV(&builder, "Model", report.Model)
	writeKV(&builder, "Context", report.ContextRelPath)
	if len(report.SourcePaths) > 0 {
		builder.WriteString("\n## Sources\n\n")
		for _, source := range report.SourcePaths {
			builder.WriteString("- ")
			builder.WriteString(source)
			builder.WriteString("\n")
		}
	}
	if len(report.CitationRefs) > 0 {
		builder.WriteString("\n## Citations\n\n")
		for _, citation := range report.CitationRefs {
			builder.WriteString("- ")
			builder.WriteString(citation)
			builder.WriteString("\n")
		}
	}
	if snippets := cleanChatAnswerSnippets(report.CitationSnippets); len(snippets) > 0 {
		builder.WriteString("\n## Citation Snippets\n\n")
		for _, snippet := range snippets {
			builder.WriteString("- ")
			builder.WriteString(snippet)
			builder.WriteString("\n")
		}
	}
	if strings.TrimSpace(report.EvidenceSummary) != "" || strings.TrimSpace(report.EvidenceQuality) != "" {
		builder.WriteString("\n## Evidence\n\n")
		writeKV(&builder, "Quality", report.EvidenceQuality)
		writeKV(&builder, "Summary", report.EvidenceSummary)
	}
	if strings.TrimSpace(report.Prompt) != "" {
		builder.WriteString("\n## Prompt\n\n")
		builder.WriteString(strings.TrimSpace(report.Prompt))
		builder.WriteString("\n")
	}
	builder.WriteString("\n## Answer\n\n")
	builder.WriteString(content)
	builder.WriteString("\n")
	return builder.String()
}

func chatAnswerTitle(prompt string) string {
	prompt = strings.Join(strings.Fields(prompt), " ")
	if prompt == "" {
		return "Assistant Answer"
	}
	if len(prompt) > 64 {
		prompt = prompt[:61] + "..."
	}
	return "Assistant Answer - " + prompt
}

func cleanChatAnswerSnippets(snippets []string) []string {
	cleaned := make([]string, 0, len(snippets))
	seen := map[string]bool{}
	for _, snippet := range snippets {
		if len(cleaned) >= chatAnswerCitationSnippetLimit {
			break
		}
		snippet = strings.Join(strings.Fields(snippet), " ")
		snippet = truncateChatAnswerSnippet(snippet)
		if snippet == "" || seen[snippet] {
			continue
		}
		seen[snippet] = true
		cleaned = append(cleaned, snippet)
	}
	return cleaned
}

func truncateChatAnswerSnippet(snippet string) string {
	runes := []rune(snippet)
	if len(runes) <= chatAnswerCitationSnippetMaxRunes {
		return snippet
	}
	return string(runes[:chatAnswerCitationSnippetMaxRunes-3]) + "..."
}

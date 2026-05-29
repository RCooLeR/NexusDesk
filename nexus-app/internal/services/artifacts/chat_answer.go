package artifacts

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const chatAnswerCitationSnippetLimit = 16
const chatAnswerCitationSnippetMaxRunes = 512
const chatAnswerSourceCoverageLimit = 64

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
	markdown := chatAnswerMarkdown(report, title, content, createdAt)
	relPath, absPath, file, err := s.createUniqueArtifactFile("chat-answers", title, ".md", createdAt)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	if _, err := file.WriteString(markdown); err != nil {
		return Artifact{}, err
	}
	metadata := Metadata{
		Kind:                   "chat-answer",
		Title:                  title,
		RelPath:                relPath,
		Source:                 firstNonEmptyArtifact(report.Source, "Nexus assistant"),
		ContextRelPath:         strings.TrimSpace(report.ContextRelPath),
		Prompt:                 strings.TrimSpace(report.Prompt),
		Model:                  strings.TrimSpace(report.Model),
		ModelRouteID:           strings.TrimSpace(report.ModelRouteID),
		ModelRoute:             strings.TrimSpace(report.ModelRoute),
		SourcePaths:            append([]string{}, report.SourcePaths...),
		CitationRefs:           cleanChatAnswerList(report.CitationRefs, chatAnswerCitationSnippetLimit),
		UnverifiedCitationRefs: cleanChatAnswerList(report.UnverifiedCitationRefs, chatAnswerCitationSnippetLimit),
		CitationSnippets:       cleanChatAnswerSnippets(report.CitationSnippets),
		CitedSourcePaths:       cleanChatAnswerList(report.CitedSourcePaths, chatAnswerSourceCoverageLimit),
		UncitedSourcePaths:     cleanChatAnswerList(report.UncitedSourcePaths, chatAnswerSourceCoverageLimit),
		EvidenceQuality:        strings.TrimSpace(report.EvidenceQuality),
		EvidenceSummary:        strings.TrimSpace(report.EvidenceSummary),
		GeneratedAt:            createdAt,
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
	writeKV(&builder, "Model route", report.ModelRoute)
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
		for _, citation := range cleanChatAnswerList(report.CitationRefs, chatAnswerCitationSnippetLimit) {
			builder.WriteString("- ")
			builder.WriteString(citation)
			builder.WriteString("\n")
		}
	}
	if unverified := cleanChatAnswerList(report.UnverifiedCitationRefs, chatAnswerCitationSnippetLimit); len(unverified) > 0 {
		builder.WriteString("\n## Unverified Citations\n\n")
		builder.WriteString("These line references were present in the answer but were not inside the attached source set.\n\n")
		for _, citation := range unverified {
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
	citedSources := cleanChatAnswerList(report.CitedSourcePaths, chatAnswerSourceCoverageLimit)
	uncitedSources := cleanChatAnswerList(report.UncitedSourcePaths, chatAnswerSourceCoverageLimit)
	if len(citedSources) > 0 || len(uncitedSources) > 0 {
		builder.WriteString("\n## Source Coverage\n\n")
		writeKV(&builder, "Cited sources", fmt.Sprintf("%d", len(citedSources)))
		writeKV(&builder, "Uncited sources", fmt.Sprintf("%d", len(uncitedSources)))
		if len(citedSources) > 0 {
			builder.WriteString("\n### Cited Sources\n\n")
			for _, source := range citedSources {
				builder.WriteString("- ")
				builder.WriteString(source)
				builder.WriteString("\n")
			}
		}
		if len(uncitedSources) > 0 {
			builder.WriteString("\n### Uncited Sources\n\n")
			for _, source := range uncitedSources {
				builder.WriteString("- ")
				builder.WriteString(source)
				builder.WriteString("\n")
			}
		}
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

func ExtractChatAnswerContent(markdown string) string {
	text := strings.ReplaceAll(markdown, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	marker := "## Answer\n"
	index := strings.LastIndex(text, "\n"+marker)
	if index >= 0 {
		index++
	} else if strings.HasPrefix(text, marker) {
		index = 0
	}
	if index < 0 {
		return strings.TrimSpace(markdown)
	}
	answer := text[index+len(marker):]
	if nextHeading := strings.Index(answer, "\n## "); nextHeading >= 0 {
		answer = answer[:nextHeading]
	}
	return strings.TrimSpace(answer)
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

func cleanChatAnswerList(values []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	cleaned := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		if len(cleaned) >= limit {
			break
		}
		value = strings.Join(strings.Fields(value), " ")
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		cleaned = append(cleaned, value)
	}
	return cleaned
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

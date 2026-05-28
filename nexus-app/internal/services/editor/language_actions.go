package editor

import (
	"fmt"
	"strings"
)

const (
	LanguageActionAvailable   = "available"
	LanguageActionFallback    = "fallback"
	LanguageActionPlanned     = "planned"
	LanguageActionUnavailable = "unavailable"
)

type LanguageAction struct {
	Name   string
	Status string
	Detail string
}

type LanguageActionPlan struct {
	Language SyntaxLanguage
	Actions  []LanguageAction
	Summary  string
}

func BuildLanguageActionPlan(fileName string, content string) LanguageActionPlan {
	language := DetectSyntaxLanguage(fileName)
	outline := BuildOutline(fileName, content)
	actions := []LanguageAction{
		{
			Name:   "Edit source",
			Status: LanguageActionAvailable,
			Detail: "Native Fyne text editing with dirty-close safety and safe-write rollback snapshots.",
		},
	}
	if language.NativeLight {
		actions = append(actions, LanguageAction{
			Name:   "Syntax highlighting",
			Status: LanguageActionAvailable,
			Detail: "Live native TextGrid mirror uses the lightweight tokenizer while editing stays in the safe text editor.",
		})
	} else {
		actions = append(actions, LanguageAction{
			Name:   "Syntax highlighting",
			Status: LanguageActionUnavailable,
			Detail: "Plain-text mode; no tokenizer is registered for this file type yet.",
		})
	}
	if CanFormatDocument(fileName) {
		actions = append(actions, LanguageAction{
			Name:   "Format draft",
			Status: LanguageActionAvailable,
			Detail: "Format runs locally on the draft before Save applies through the safe-write service.",
		})
	} else {
		actions = append(actions, LanguageAction{
			Name:   "Format draft",
			Status: LanguageActionUnavailable,
			Detail: "No native formatter or whitespace strategy is registered for this file type.",
		})
	}
	if len(outline) > 0 {
		actions = append(actions,
			LanguageAction{
				Name:   "Outline and symbols",
				Status: LanguageActionAvailable,
				Detail: fmt.Sprintf("%d native symbol(s) detected for jump navigation.", len(outline)),
			},
			LanguageAction{
				Name:   "Definition and references",
				Status: LanguageActionFallback,
				Detail: "Native outline plus bounded workspace search provide local navigation without starting an external language server.",
			},
		)
	} else if language.FutureLSP {
		actions = append(actions,
			LanguageAction{
				Name:   "Outline and symbols",
				Status: LanguageActionPlanned,
				Detail: "No native symbols detected; this language remains a candidate for future LSP-backed symbol indexing.",
			},
			LanguageAction{
				Name:   "Definition and references",
				Status: LanguageActionPlanned,
				Detail: "Future LSP integration should replace the current bounded text-search fallback for this language.",
			},
		)
	} else {
		actions = append(actions,
			LanguageAction{
				Name:   "Outline and symbols",
				Status: LanguageActionUnavailable,
				Detail: "No symbols detected by the native outline rules.",
			},
			LanguageAction{
				Name:   "Definition and references",
				Status: LanguageActionUnavailable,
				Detail: "No symbol index is available for this file type.",
			},
		)
	}
	if language.FutureLSP {
		actions = append(actions, LanguageAction{
			Name:   "External LSP",
			Status: LanguageActionPlanned,
			Detail: "Language is marked as an LSP candidate; native editing remains framework-free until packaging and accessibility are proven.",
		})
	}
	return LanguageActionPlan{
		Language: language,
		Actions:  actions,
		Summary:  languageActionSummary(actions),
	}
}

func languageActionSummary(actions []LanguageAction) string {
	counts := map[string]int{}
	for _, action := range actions {
		status := strings.TrimSpace(action.Status)
		if status == "" {
			status = LanguageActionUnavailable
		}
		counts[status]++
	}
	parts := []string{}
	for _, status := range []string{LanguageActionAvailable, LanguageActionFallback, LanguageActionPlanned, LanguageActionUnavailable} {
		if counts[status] > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", status, counts[status]))
		}
	}
	if len(parts) == 0 {
		return "no language actions"
	}
	return strings.Join(parts, ", ")
}

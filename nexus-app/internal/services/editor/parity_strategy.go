package editor

import "strings"

type EditorParityStrategy struct {
	Status         string
	Decision       string
	InlineSyntax   string
	Navigation     string
	Diagnostics    string
	LSP            string
	BetaBlocker    bool
	NextMilestones []string
}

func NativeParityEditorStrategy() EditorParityStrategy {
	return EditorParityStrategy{
		Status:       "accepted-for-native-parity-beta",
		Decision:     "Use the native Fyne text editor for safe editing, with the Syntax mirror and Document Map as the Native Parity Beta replacement for Monaco minimap and inline token styling.",
		InlineSyntax: "Do not block Native Parity Beta on active-editor inline token styling; keep read-only TextGrid highlighting and cursor-aware token status as the production-safe baseline.",
		Navigation:   "Use native outline, go-to-symbol, local definition, bounded workspace definition fallback, and bounded references search until a packaged LSP provider is proven.",
		Diagnostics:  "Use live draft diagnostics plus saved-file Problems syntax scans for local parser coverage.",
		LSP:          "Treat LSP as a post-beta enhancement that must prove packaging, accessibility, cancellation, and failure isolation before becoming default.",
		BetaBlocker:  false,
		NextMilestones: []string{
			"spike editable-widget inline styling only if it preserves safe editing and accessibility",
			"prototype one packaged LSP provider behind an explicit feature flag",
			"expand semantic diagnostics after local syntax scans remain stable",
		},
	}
}

func (s EditorParityStrategy) Summary() string {
	parts := []string{}
	for _, part := range []string{s.Status, s.Decision, s.LSP} {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "No native editor parity strategy recorded."
	}
	return strings.Join(parts, " ")
}

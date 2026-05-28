package editor

import (
	"strings"
	"testing"
)

func TestBuildLanguageActionPlanReportsNativeGoActions(t *testing.T) {
	plan := BuildLanguageActionPlan("main.go", "package main\n\nfunc main() {}\n")
	if plan.Language.ID != "go" {
		t.Fatalf("expected Go language, got %#v", plan.Language)
	}
	if !strings.Contains(plan.Summary, "available=") || !strings.Contains(plan.Summary, "fallback=") || !strings.Contains(plan.Summary, "planned=") {
		t.Fatalf("expected available/fallback/planned summary, got %q", plan.Summary)
	}
	assertLanguageAction(t, plan, "Syntax highlighting", LanguageActionAvailable)
	assertLanguageAction(t, plan, "Format draft", LanguageActionAvailable)
	assertLanguageAction(t, plan, "Draft diagnostics", LanguageActionAvailable)
	assertLanguageAction(t, plan, "Outline and symbols", LanguageActionAvailable)
	assertLanguageAction(t, plan, "Definition and references", LanguageActionFallback)
	assertLanguageAction(t, plan, "External LSP", LanguageActionPlanned)
	assertLanguageAction(t, plan, "Native Parity Beta strategy", LanguageActionAvailable)
	if plan.BetaStrategy.BetaBlocker {
		t.Fatal("native editor beta strategy should not block Native Parity Beta")
	}
}

func TestBuildLanguageActionPlanReportsPlainTextLimits(t *testing.T) {
	plan := BuildLanguageActionPlan("notes.unknown", "just text\n")
	assertLanguageAction(t, plan, "Syntax highlighting", LanguageActionUnavailable)
	assertLanguageAction(t, plan, "Format draft", LanguageActionUnavailable)
	assertLanguageAction(t, plan, "Draft diagnostics", LanguageActionUnavailable)
	assertLanguageAction(t, plan, "Outline and symbols", LanguageActionUnavailable)
	assertLanguageAction(t, plan, "Definition and references", LanguageActionUnavailable)
	if strings.Contains(plan.Summary, "planned=") {
		t.Fatalf("did not expect planned LSP action for plain text: %q", plan.Summary)
	}
}

func TestCanFormatDocumentMatchesSupportedFormats(t *testing.T) {
	for _, fileName := range []string{"main.go", "settings.json", "README.md", "compose.yaml", "Dockerfile.dev", "script.py", "style.css"} {
		if !CanFormatDocument(fileName) {
			t.Fatalf("expected %s to be format-capable", fileName)
		}
	}
	if CanFormatDocument("archive.bin") {
		t.Fatal("did not expect binary extension to be format-capable")
	}
}

func assertLanguageAction(t *testing.T, plan LanguageActionPlan, name string, status string) {
	t.Helper()
	for _, action := range plan.Actions {
		if action.Name == name {
			if action.Status != status {
				t.Fatalf("action %q status = %q, want %q", name, action.Status, status)
			}
			if strings.TrimSpace(action.Detail) == "" {
				t.Fatalf("action %q should include detail", name)
			}
			return
		}
	}
	t.Fatalf("action %q not found in %#v", name, plan.Actions)
}

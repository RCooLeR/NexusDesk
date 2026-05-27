package shell

import (
	"strings"
	"testing"

	operationsSvc "nexusdesk/internal/services/operations"
)

func TestFormatOperationsInspectionIncludesServicesAndWarnings(t *testing.T) {
	text := formatOperationsInspection(operationsSvc.Inspection{
		File: operationsSvc.File{RelPath: "compose.yml", Kind: operationsSvc.FileKindCompose, Size: 42},
		Text: "services:\n  api:\n    image: app\n",
		Services: []operationsSvc.ComposeService{
			{Name: "api", Image: "app", Ports: []string{"8080:80"}, DependsOn: []string{"db"}},
		},
		Warnings: []string{"Read-only inspection only."},
	})
	for _, want := range []string{"# Operations Inspection", "compose.yml", "Compose Services", "api | image: app", "Read-only inspection only."} {
		if !strings.Contains(text, want) {
			t.Fatalf("formatted inspection missing %q:\n%s", want, text)
		}
	}
}

func TestFormatOperationsFileLabelShowsKind(t *testing.T) {
	label := formatOperationsFileLabel(operationsSvc.File{RelPath: ".env", Kind: operationsSvc.FileKindEnv, Size: 12})
	if !strings.Contains(label, ".env") || !strings.Contains(label, "env") {
		t.Fatalf("unexpected label: %s", label)
	}
}

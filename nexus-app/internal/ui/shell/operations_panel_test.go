package shell

import (
	"context"
	"strings"
	"testing"

	operationsSvc "nexusdesk/internal/services/operations"
)

func TestNewOperationsControllerOwnsPanelState(t *testing.T) {
	view := &View{}
	controller := newOperationsController(view)
	if controller.view != view {
		t.Fatal("expected operations controller to keep owning view reference")
	}
	if controller.results == nil || controller.status == nil || controller.detail == nil {
		t.Fatal("expected operations controller to initialize panel widgets")
	}
	if controller.status.Text != "Operations scan has not been run." {
		t.Fatalf("unexpected initial operations status: %q", controller.status.Text)
	}
}

func TestFormatOperationsInspectionIncludesServicesAndWarnings(t *testing.T) {
	text := formatOperationsInspection(operationsSvc.Inspection{
		File: operationsSvc.File{RelPath: "compose.yml", Kind: operationsSvc.FileKindCompose, Size: 42},
		Text: "services:\n  api:\n    image: app\n",
		Services: []operationsSvc.ComposeService{
			{Name: "api", Image: "app", Ports: []string{"8080:80"}, DependsOn: []string{"db"}},
		},
		Topology: operationsSvc.ComposeTopology{
			Summary: "1 service(s), 1 dependency edge(s), 1 exposed port(s), 0 named volume(s).",
			Edges: []operationsSvc.ComposeTopologyEdge{
				{From: "api", To: "db", Relation: "depends_on", Missing: true},
			},
			ExposedPorts: []operationsSvc.ComposePortExposure{{Service: "api", Port: "8080:80"}},
			Warnings:     []string{"Service \"api\" depends on \"db\", but that service was not found."},
		},
		Warnings: []string{"Read-only inspection only."},
	})
	for _, want := range []string{"# Operations Inspection", "compose.yml", "Compose Services", "api | image: app", "Compose Topology", "api -> db", "api exposes 8080:80", "Read-only inspection only."} {
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

func TestOperationsRunbookArtifactInputPreservesEvidence(t *testing.T) {
	input := operationsRunbookArtifactInput(operationsSvc.Inspection{
		File: operationsSvc.File{RelPath: "compose.yml", Name: "compose.yml", Kind: operationsSvc.FileKindCompose, Size: 42},
		Text: "services:\n  api:\n    image: app\n",
		Services: []operationsSvc.ComposeService{
			{Name: "api", Image: "app", Ports: []string{"8080:80"}},
		},
		Topology: operationsSvc.ComposeTopology{
			Summary:      "1 service(s), 0 dependency edge(s), 1 exposed port(s), 0 named volume(s).",
			ExposedPorts: []operationsSvc.ComposePortExposure{{Service: "api", Port: "8080:80"}},
		},
		Warnings: []string{"Read-only inspection only."},
	})
	if input.SourcePath != "compose.yml" || input.Kind != "compose" || len(input.Services) != 1 || input.TopologySummary == "" || len(input.ExposedPorts) != 1 {
		t.Fatalf("unexpected operations runbook input: %#v", input)
	}
	for _, want := range []string{"# Operations Inspection", "compose.yml", "Read-only inspection only."} {
		if !strings.Contains(input.Content, want) {
			t.Fatalf("runbook content missing %q:\n%s", want, input.Content)
		}
	}
}

func TestOperationsJobLabels(t *testing.T) {
	if got := operationsScanJobLabel(); got != "Operations scan" {
		t.Fatalf("unexpected scan label: %q", got)
	}
	if got := operationsInspectJobLabel("docker-compose.yml"); got != "Operations inspect (docker-compose.yml)" {
		t.Fatalf("unexpected inspect label: %q", got)
	}
	if got := operationsRunbookJobLabel("compose.yml"); got != "Operations runbook export (compose.yml)" {
		t.Fatalf("unexpected runbook label: %q", got)
	}
}

func TestIsOperationsJobCanceled(t *testing.T) {
	if !isOperationsJobCanceled(context.Canceled) {
		t.Fatal("expected context.Canceled to be treated as canceled job")
	}
	if !isOperationsJobCanceled(context.DeadlineExceeded) {
		t.Fatal("expected context.DeadlineExceeded to be treated as canceled job")
	}
}

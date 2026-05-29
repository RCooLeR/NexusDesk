package readiness

import (
	"errors"
	"fmt"
	"strings"
)

type FailureScenario struct {
	ID                string
	Label             string
	Owner             string
	Risk              string
	ExpectedBehavior  string
	AutomatedCoverage []string
	ManualCoverage    []string
}

func ProductionFailureScenarios() []FailureScenario {
	scenarios := []FailureScenario{
		{
			ID:               "folder-open-cheap",
			Label:            "Folder open does not trigger expensive work",
			Owner:            "workspace/readiness/jobs",
			Risk:             "workspace-open hang or surprise external work",
			ExpectedBehavior: "Opening a workspace performs only bounded local listing/readiness work; Git, Docker, OCR, connector pulls, model calls, shell, browser automation, dump imports, and deep indexing stay explicit user actions.",
			AutomatedCoverage: []string{
				"internal/services/workspace: TestOpenListsOnlyTopLevel",
				"internal/services/jobs: TestSlowWorkflowSpecsRequireDurableExplicitStarts",
				"internal/services/readiness: TestFailureScenariosCoverProductionGate",
			},
			ManualCoverage: []string{
				"docs/05_PLAN.md: launch/open workspace smoke",
			},
		},
		{
			ID:               "malformed-files",
			Label:            "Malformed or unsupported files fail safely",
			Owner:            "workspace/documents/datasets",
			Risk:             "preview crash, memory spike, or silent bad decode",
			ExpectedBehavior: "Preview/extraction paths cap reads, return user-visible errors or safe truncated previews, and never mutate source files.",
			AutomatedCoverage: []string{
				"internal/services/workspace: TestPreviewFileRejectsOversizedBinaryFiles",
				"internal/services/workspace: TestPreviewFileMarksBinaryWithoutText",
				"internal/services/workspace: TestPreviewFileRejectsTraversal",
			},
			ManualCoverage: []string{
				"docs/05_PLAN.md: malformed file preview smoke",
			},
		},
		{
			ID:               "corrupt-metadata",
			Label:            "Corrupt metadata recovers or reports clearly",
			Owner:            "metadata/diagnostics/workspace",
			Risk:             "workspace cannot open or history silently disappears",
			ExpectedBehavior: "Corrupt metadata is quarantined or reported through Diagnostics without blocking normal workspace inspection.",
			AutomatedCoverage: []string{
				"internal/services/metadata: corrupt metadata archive tests",
				"internal/services/workspace: TestWriteSearchMetadataQuarantinesCorruptManifest",
				"internal/ui/shell: TestDiagnosticsHealthCardsSummarizeActionsAndWarnings",
			},
			ManualCoverage: []string{
				"docs/05_PLAN.md: diagnostics/metadata smoke",
			},
		},
		{
			ID:               "missing-provider",
			Label:            "Missing provider/model is actionable",
			Owner:            "settings/llm/readiness/diagnostics",
			Risk:             "assistant appears broken with no clear recovery path",
			ExpectedBehavior: "Home readiness, Settings, and Diagnostics show provider/model/API-key gaps and recovery guidance before long agent work starts.",
			AutomatedCoverage: []string{
				"internal/services/readiness: TestCollectFlagsFirstRunActions",
				"internal/services/readiness: TestCollectRequiresAPIKeyForCustomProvider",
				"internal/ui/shell: provider diagnostics guidance tests",
			},
			ManualCoverage: []string{
				"docs/05_PLAN.md: provider setup smoke",
			},
		},
		{
			ID:               "canceled-long-work",
			Label:            "Canceled long work records safe state",
			Owner:            "jobs/tasks/datasets/artifacts/operations",
			Risk:             "partial output, stuck job, or confusing retry state",
			ExpectedBehavior: "Cancelable work exits promptly, records canceled status/logs, preserves safe outputs only when intended, and remains inspectable from Jobs/Diagnostics.",
			AutomatedCoverage: []string{
				"internal/services/tasks: canceled task run tests",
				"internal/services/datasets: canceled notebook/query tests",
				"internal/services/workspace: TestScanReportHonorsCanceledContext",
				"internal/services/tools: cancel_job approval/cancellation tests",
			},
			ManualCoverage: []string{
				"docs/05_PLAN.md: cancel/retry smoke",
			},
		},
	}
	return cloneFailureScenarios(scenarios)
}

func ValidateProductionFailureScenarios(scenarios []FailureScenario) error {
	if len(scenarios) == 0 {
		return errors.New("production failure scenario matrix is empty")
	}
	seen := map[string]bool{}
	for _, scenario := range scenarios {
		if strings.TrimSpace(scenario.ID) == "" {
			return errors.New("production failure scenario is missing an id")
		}
		if seen[scenario.ID] {
			return fmt.Errorf("duplicate production failure scenario id %q", scenario.ID)
		}
		seen[scenario.ID] = true
		if strings.TrimSpace(scenario.Label) == "" || strings.TrimSpace(scenario.Owner) == "" || strings.TrimSpace(scenario.ExpectedBehavior) == "" {
			return fmt.Errorf("production failure scenario %q is incomplete", scenario.ID)
		}
		if len(scenario.AutomatedCoverage) == 0 {
			return fmt.Errorf("production failure scenario %q has no automated coverage", scenario.ID)
		}
		if len(scenario.ManualCoverage) == 0 {
			return fmt.Errorf("production failure scenario %q has no manual smoke coverage", scenario.ID)
		}
	}
	return nil
}

func FormatFailureScenarioMatrix(scenarios []FailureScenario) string {
	if len(scenarios) == 0 {
		return "No production failure scenarios are registered."
	}
	var builder strings.Builder
	builder.WriteString("## Production failure scenarios\n\n")
	for _, scenario := range scenarios {
		builder.WriteString("- **")
		builder.WriteString(scenario.Label)
		builder.WriteString("** (`")
		builder.WriteString(scenario.ID)
		builder.WriteString("`): ")
		builder.WriteString(scenario.ExpectedBehavior)
		builder.WriteString(" Owner: ")
		builder.WriteString(scenario.Owner)
		builder.WriteString(". Automated: ")
		builder.WriteString(strings.Join(scenario.AutomatedCoverage, "; "))
		builder.WriteString(". Manual: ")
		builder.WriteString(strings.Join(scenario.ManualCoverage, "; "))
		builder.WriteString(".\n")
	}
	return builder.String()
}

func cloneFailureScenarios(scenarios []FailureScenario) []FailureScenario {
	out := make([]FailureScenario, len(scenarios))
	for index, scenario := range scenarios {
		out[index] = scenario
		out[index].AutomatedCoverage = cloneStrings(scenario.AutomatedCoverage)
		out[index].ManualCoverage = cloneStrings(scenario.ManualCoverage)
	}
	return out
}

func cloneStrings(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}

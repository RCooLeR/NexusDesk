package readiness

import (
	"strings"
	"testing"
)

func TestFailureScenariosCoverProductionGate(t *testing.T) {
	scenarios := ProductionFailureScenarios()
	if err := ValidateProductionFailureScenarios(scenarios); err != nil {
		t.Fatalf("failure scenario matrix is invalid: %v", err)
	}
	byID := map[string]FailureScenario{}
	for _, scenario := range scenarios {
		byID[scenario.ID] = scenario
	}
	for _, id := range []string{
		"folder-open-cheap",
		"malformed-files",
		"corrupt-metadata",
		"missing-provider",
		"canceled-long-work",
	} {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing production failure scenario %q", id)
		}
	}
}

func TestValidateProductionFailureScenariosRejectsDrift(t *testing.T) {
	if err := ValidateProductionFailureScenarios([]FailureScenario{{
		ID:               "missing-coverage",
		Label:            "Missing coverage",
		Owner:            "readiness",
		ExpectedBehavior: "should fail validation",
	}}); err == nil {
		t.Fatal("expected validation to reject missing coverage")
	}
	if err := ValidateProductionFailureScenarios([]FailureScenario{
		{ID: "dup", Label: "A", Owner: "readiness", ExpectedBehavior: "A", AutomatedCoverage: []string{"test"}, ManualCoverage: []string{"smoke"}},
		{ID: "dup", Label: "B", Owner: "readiness", ExpectedBehavior: "B", AutomatedCoverage: []string{"test"}, ManualCoverage: []string{"smoke"}},
	}); err == nil {
		t.Fatal("expected validation to reject duplicate ids")
	}
}

func TestFormatFailureScenarioMatrixIncludesEvidence(t *testing.T) {
	text := FormatFailureScenarioMatrix(ProductionFailureScenarios())
	for _, expected := range []string{
		"Production failure scenarios",
		"folder-open-cheap",
		"TestOpenListsOnlyTopLevel",
		"docs/05_PLAN.md",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("failure scenario matrix missing %q:\n%s", expected, text)
		}
	}
}

func TestProductionFailureScenariosReturnsCopies(t *testing.T) {
	scenarios := ProductionFailureScenarios()
	scenarios[0].AutomatedCoverage[0] = "mutated"
	if ProductionFailureScenarios()[0].AutomatedCoverage[0] == "mutated" {
		t.Fatal("ProductionFailureScenarios leaked mutable coverage slices")
	}
}

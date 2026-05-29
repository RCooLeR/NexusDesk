package security

import "testing"

func TestControlsForRiskRequiresApprovalForRiskyTools(t *testing.T) {
	for _, risk := range []string{"medium", "high"} {
		controls := ControlsForRisk(risk)
		for _, required := range []string{ControlApproval, ControlAudit, ControlTimeout, ControlCancellation, ControlOutputCap, ControlRedaction} {
			if !hasControl(controls, required) {
				t.Fatalf("%s risk controls missing %s: %#v", risk, required, controls)
			}
		}
	}
	if hasControl(ControlsForRisk("low"), ControlApproval) {
		t.Fatal("low-risk controls should not require approval by default")
	}
}

func TestThreatFamiliesCoverFutureHighRiskWorkflows(t *testing.T) {
	families := ThreatFamilies()
	byID := map[string]ThreatFamily{}
	for _, family := range families {
		byID[family.ID] = family
	}
	for _, id := range []string{
		"filesystem",
		"connectors",
		"jobs",
		"terminal-shell",
		"browser-automation",
		"docker-system",
		"generated-artifacts",
		"mcp-plugins",
	} {
		family, ok := byID[id]
		if !ok {
			t.Fatalf("missing threat family %q", id)
		}
		if family.Name == "" || family.Description == "" || len(family.Controls) == 0 {
			t.Fatalf("incomplete threat family %q: %#v", id, family)
		}
	}
}

func TestHighImpactFamiliesRequireDurableAndVisibleControls(t *testing.T) {
	for _, id := range []string{"jobs", "terminal-shell", "docker-system", "browser-automation"} {
		controls := RequiredControlsForFamilies(id)
		for _, required := range []string{ControlAudit, ControlTimeout, ControlCancellation, ControlOutputCap, ControlNoWorkspaceOpen} {
			if !hasControl(controls, required) {
				t.Fatalf("%s controls missing %s: %#v", id, required, controls)
			}
		}
	}
	for _, id := range []string{"terminal-shell", "docker-system", "browser-automation"} {
		if !hasControl(RequiredControlsForFamilies(id), ControlApproval) {
			t.Fatalf("%s controls must require approval", id)
		}
	}
}

func TestReturnedThreatFamiliesAreImmutableCopies(t *testing.T) {
	families := ThreatFamilies()
	families[0].Controls[0] = "mutated"
	if ThreatFamilies()[0].Controls[0] == "mutated" {
		t.Fatal("ThreatFamilies leaked mutable control slices")
	}
}

func hasControl(controls []string, required string) bool {
	for _, control := range controls {
		if control == required {
			return true
		}
	}
	return false
}

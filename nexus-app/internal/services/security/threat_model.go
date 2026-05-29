package security

import "sort"

const (
	ControlApproval          = "approval"
	ControlAudit             = "audit"
	ControlRollback          = "rollback-or-mitigation"
	ControlRedaction         = "redaction"
	ControlRootedScope       = "rooted-scope"
	ControlConnectorScope    = "connector-scope"
	ControlTimeout           = "timeout"
	ControlCancellation      = "cancellation"
	ControlOutputCap         = "output-cap"
	ControlDurableJob        = "durable-job"
	ControlSecretIsolation   = "secret-isolation"
	ControlArtifactLineage   = "artifact-lineage"
	ControlSandbox           = "sandbox"
	ControlNoWorkspaceOpen   = "no-workspace-open"
	ControlPreview           = "preview"
	ControlUserVisibleStatus = "user-visible-status"
)

type ThreatFamily struct {
	ID          string
	Name        string
	Description string
	Controls    []string
}

var baseRiskControls = map[string][]string{
	"low":    {ControlRootedScope, ControlOutputCap},
	"medium": {ControlRootedScope, ControlApproval, ControlAudit, ControlTimeout, ControlCancellation, ControlOutputCap, ControlRedaction},
	"high":   {ControlRootedScope, ControlApproval, ControlAudit, ControlRollback, ControlTimeout, ControlCancellation, ControlOutputCap, ControlRedaction, ControlUserVisibleStatus},
}

var threatFamilies = []ThreatFamily{
	{
		ID:          "filesystem",
		Name:        "Workspace filesystem",
		Description: "Workspace reads, writes, patches, deletes, generated files, and rollback-backed file mutations.",
		Controls:    []string{ControlRootedScope, ControlApproval, ControlAudit, ControlRollback, ControlPreview, ControlRedaction, ControlOutputCap},
	},
	{
		ID:          "connectors",
		Name:        "External connectors and databases",
		Description: "Database profiles, analytics/CRM/cloud connectors, connector pulls, and remote API actions.",
		Controls:    []string{ControlConnectorScope, ControlApproval, ControlAudit, ControlDurableJob, ControlSecretIsolation, ControlTimeout, ControlCancellation, ControlRedaction, ControlArtifactLineage},
	},
	{
		ID:          "jobs",
		Name:        "Durable slow workflows",
		Description: "OCR, dump imports, connector pulls, report generation, long indexing, long agent runs, and packaged exports.",
		Controls:    []string{ControlDurableJob, ControlNoWorkspaceOpen, ControlUserVisibleStatus, ControlAudit, ControlTimeout, ControlCancellation, ControlOutputCap, ControlArtifactLineage},
	},
	{
		ID:          "terminal-shell",
		Name:        "Terminal and shell execution",
		Description: "Approved one-shot terminal commands and future interactive terminal sessions.",
		Controls:    []string{ControlApproval, ControlAudit, ControlDurableJob, ControlRootedScope, ControlTimeout, ControlCancellation, ControlOutputCap, ControlRedaction, ControlNoWorkspaceOpen},
	},
	{
		ID:          "browser-automation",
		Name:        "Rendered browser automation",
		Description: "Browser sessions, navigation, clicks, typing, screenshots, page extraction, and network logs.",
		Controls:    []string{ControlApproval, ControlAudit, ControlSandbox, ControlTimeout, ControlCancellation, ControlOutputCap, ControlRedaction, ControlArtifactLineage, ControlNoWorkspaceOpen},
	},
	{
		ID:          "docker-system",
		Name:        "Docker and system operations",
		Description: "Docker Compose config/logs/lifecycle, containers, volumes, images, and future system-impacting operations.",
		Controls:    []string{ControlApproval, ControlAudit, ControlDurableJob, ControlSandbox, ControlTimeout, ControlCancellation, ControlOutputCap, ControlRedaction, ControlUserVisibleStatus, ControlNoWorkspaceOpen},
	},
	{
		ID:          "generated-artifacts",
		Name:        "Generated artifacts and media",
		Description: "Reports, charts, DOCX, PPTX, images, exports, regenerated artifacts, and packaged outputs.",
		Controls:    []string{ControlApproval, ControlAudit, ControlArtifactLineage, ControlPreview, ControlOutputCap, ControlRedaction, ControlDurableJob},
	},
	{
		ID:          "mcp-plugins",
		Name:        "MCP and plugins",
		Description: "Third-party tool discovery, MCP tool calls, plugin installation, signed extension lifecycle, and permissioned execution.",
		Controls:    []string{ControlApproval, ControlAudit, ControlSandbox, ControlConnectorScope, ControlSecretIsolation, ControlTimeout, ControlCancellation, ControlOutputCap, ControlRedaction, ControlNoWorkspaceOpen},
	},
}

func ControlsForRisk(risk string) []string {
	controls, ok := baseRiskControls[risk]
	if !ok {
		return []string{ControlApproval, ControlAudit, ControlOutputCap}
	}
	return cloneStrings(controls)
}

func ThreatFamilies() []ThreatFamily {
	out := make([]ThreatFamily, len(threatFamilies))
	for index, family := range threatFamilies {
		out[index] = family
		out[index].Controls = cloneStrings(family.Controls)
	}
	return out
}

func RequiredControlsForFamilies(ids ...string) []string {
	controls := map[string]bool{}
	for _, family := range threatFamilies {
		if !containsString(ids, family.ID) {
			continue
		}
		for _, control := range family.Controls {
			controls[control] = true
		}
	}
	out := make([]string, 0, len(controls))
	for control := range controls {
		out = append(out, control)
	}
	sort.Strings(out)
	return out
}

func cloneStrings(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

package jobs

import (
	"errors"
	"fmt"
	"strings"
)

const (
	KindTask                 = "task"
	KindMetadataCompatImport = "metadata-compat-import"
	KindOCRExtraction        = "ocr-extraction"
	KindDumpImport           = "dump-import"
	KindConnectorPull        = "connector-pull"
	KindLongIndexing         = "long-indexing"
	KindReportGeneration     = "report-generation"
	KindLongAgentRun         = "long-agent-run"
	KindExternalAgentRun     = "external-agent-run"
	KindPackagedExport       = "packaged-export"
)

type WorkflowSpec struct {
	Kind                      string
	Label                     string
	Description               string
	RequiresDurableJob        bool
	ProhibitedOnWorkspaceOpen bool
	RequiresExplicitStart     bool
	Cancellable               bool
	Retryable                 bool
	AuditRequired             bool
}

type StartOptions struct {
	ExplicitUserStart bool
}

var slowWorkflowSpecs = []WorkflowSpec{
	{
		Kind:                      KindOCRExtraction,
		Label:                     "OCR extraction",
		Description:               "Extract text from scanned PDFs or images without blocking workspace open.",
		RequiresDurableJob:        true,
		ProhibitedOnWorkspaceOpen: true,
		RequiresExplicitStart:     true,
		Cancellable:               true,
		Retryable:                 true,
		AuditRequired:             true,
	},
	{
		Kind:                      KindDumpImport,
		Label:                     "Database dump import",
		Description:               "Import database dumps into isolated temporary environments for read-only analysis.",
		RequiresDurableJob:        true,
		ProhibitedOnWorkspaceOpen: true,
		RequiresExplicitStart:     true,
		Cancellable:               true,
		Retryable:                 true,
		AuditRequired:             true,
	},
	{
		Kind:                      KindConnectorPull,
		Label:                     "Connector pull",
		Description:               "Fetch remote connector data with retry, redaction, cancellation, and credential audit boundaries.",
		RequiresDurableJob:        true,
		ProhibitedOnWorkspaceOpen: true,
		RequiresExplicitStart:     true,
		Cancellable:               true,
		Retryable:                 true,
		AuditRequired:             true,
	},
	{
		Kind:                      KindLongIndexing,
		Label:                     "Long indexing",
		Description:               "Build larger search or analysis indexes after explicit user action.",
		RequiresDurableJob:        true,
		ProhibitedOnWorkspaceOpen: true,
		RequiresExplicitStart:     true,
		Cancellable:               true,
		Retryable:                 true,
		AuditRequired:             false,
	},
	{
		Kind:                      KindReportGeneration,
		Label:                     "Report generation",
		Description:               "Generate larger reports or document sets with progress, output linkage, and cancellation.",
		RequiresDurableJob:        true,
		ProhibitedOnWorkspaceOpen: true,
		RequiresExplicitStart:     true,
		Cancellable:               true,
		Retryable:                 true,
		AuditRequired:             true,
	},
	{
		Kind:                      KindLongAgentRun,
		Label:                     "Long agent run",
		Description:               "Run multi-step agent workflows through durable progress and audit records.",
		RequiresDurableJob:        true,
		ProhibitedOnWorkspaceOpen: true,
		RequiresExplicitStart:     true,
		Cancellable:               true,
		Retryable:                 false,
		AuditRequired:             true,
	},
	{
		Kind:                      KindExternalAgentRun,
		Label:                     "External coding-agent run",
		Description:               "Run optional coding-agent CLIs such as Codex, Claude Code, or OpenCode only after explicit approval.",
		RequiresDurableJob:        true,
		ProhibitedOnWorkspaceOpen: true,
		RequiresExplicitStart:     true,
		Cancellable:               true,
		Retryable:                 false,
		AuditRequired:             true,
	},
	{
		Kind:                      KindPackagedExport,
		Label:                     "Packaged export",
		Description:               "Create packaged document or presentation exports with linked artifacts and progress.",
		RequiresDurableJob:        true,
		ProhibitedOnWorkspaceOpen: true,
		RequiresExplicitStart:     true,
		Cancellable:               true,
		Retryable:                 true,
		AuditRequired:             true,
	},
}

func SlowWorkflowSpecs() []WorkflowSpec {
	out := make([]WorkflowSpec, len(slowWorkflowSpecs))
	copy(out, slowWorkflowSpecs)
	return out
}

func SlowWorkflowSpec(kind string) (WorkflowSpec, bool) {
	kind = NormalizeKind(kind)
	for _, spec := range slowWorkflowSpecs {
		if spec.Kind == kind {
			return spec, true
		}
	}
	return WorkflowSpec{}, false
}

func NormalizeKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}

func RequiresDurableJob(kind string) bool {
	spec, ok := SlowWorkflowSpec(kind)
	return ok && spec.RequiresDurableJob
}

func ProhibitedOnWorkspaceOpen(kind string) bool {
	spec, ok := SlowWorkflowSpec(kind)
	return ok && spec.ProhibitedOnWorkspaceOpen
}

func ValidateWorkflowStart(kind string, options StartOptions) error {
	spec, ok := SlowWorkflowSpec(kind)
	if !ok {
		return nil
	}
	if spec.RequiresExplicitStart && !options.ExplicitUserStart {
		return fmt.Errorf("%s jobs require an explicit user action", spec.Label)
	}
	if !spec.RequiresDurableJob {
		return errors.New("slow workflow spec must require durable job routing")
	}
	return nil
}

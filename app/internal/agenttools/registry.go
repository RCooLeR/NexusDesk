package agenttools

type Descriptor struct {
	Name             string   `json:"name"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Surface          string   `json:"surface"`
	Risk             string   `json:"risk"`
	RequiresApproval bool     `json:"requiresApproval"`
	Inputs           []string `json:"inputs"`
}

func Registry() []Descriptor {
	return []Descriptor{
		{
			Name:             "workspace.preview",
			Title:            "Preview workspace file",
			Description:      "Read a bounded preview for a file inside the approved workspace root.",
			Surface:          "Code Studio",
			Risk:             "low",
			RequiresApproval: false,
			Inputs:           []string{"relPath"},
		},
		{
			Name:             "workspace.write",
			Title:            "Write text file",
			Description:      "Create or update a text/code file through diff preview and modal approval.",
			Surface:          "Code Studio",
			Risk:             "high",
			RequiresApproval: true,
			Inputs:           []string{"relPath", "content"},
		},
		{
			Name:             "dataset.query",
			Title:            "Query CSV dataset",
			Description:      "Run a bounded CSV query with text, column, comparison, order, and limit clauses.",
			Surface:          "Data Studio",
			Risk:             "low",
			RequiresApproval: false,
			Inputs:           []string{"relPath", "query"},
		},
		{
			Name:             "artifact.create",
			Title:            "Create artifact",
			Description:      "Create deterministic reports, summaries, charts, or query exports under .nexusdesk/artifacts.",
			Surface:          "Artifact Studio",
			Risk:             "low",
			RequiresApproval: false,
			Inputs:           []string{"sourcePath", "kind"},
		},
		{
			Name:             "artifact.archive",
			Title:            "Archive artifact",
			Description:      "Move a generated artifact and its metadata sidecar into the artifact archive folder.",
			Surface:          "Artifact Studio",
			Risk:             "medium",
			RequiresApproval: true,
			Inputs:           []string{"relPath"},
		},
		{
			Name:             "operations.inspect",
			Title:            "Inspect operations files",
			Description:      "Read Dockerfile, Compose, environment, and service configuration without mutating Docker state.",
			Surface:          "Operations Studio",
			Risk:             "low",
			RequiresApproval: false,
			Inputs:           []string{"relPath"},
		},
	}
}

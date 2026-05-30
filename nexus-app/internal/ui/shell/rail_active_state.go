package shell

import "fyne.io/fyne/v2/widget"

const (
	defaultLeftRailTool  = "Project"
	defaultRightRailTool = "Assistant"
)

type railWorkspaceState struct {
	LeftTool  string
	RightTool string
}

func (v *View) setLeftRailActive(label string) {
	v.activeLeftRailTool = label
	v.rememberActiveRailTools()
	v.refreshRailActiveState()
}

func (v *View) setRightRailActive(label string) {
	v.activeRightRailTool = label
	v.rememberActiveRailTools()
	v.refreshRailActiveState()
}

func (v *View) updateRailActiveStateForTab(title string) {
	if label := leftRailLabelForTab(title); label != "" {
		v.activeLeftRailTool = label
	}
	if label := rightRailLabelForTab(title); label != "" {
		v.activeRightRailTool = label
	}
	v.rememberActiveRailTools()
	v.refreshRailActiveState()
}

func (v *View) refreshRailActiveState() {
	applyRailButtonImportance(v.leftRailButtons, v.activeLeftRailTool)
	applyRailButtonImportance(v.rightRailButtons, v.activeRightRailTool)
}

func applyRailButtonImportance(buttons map[string]*widget.Button, active string) {
	for label, button := range buttons {
		if button == nil {
			continue
		}
		if label == active {
			button.Importance = widget.HighImportance
		} else {
			button.Importance = widget.LowImportance
		}
		button.Refresh()
	}
}

func (v *View) rememberActiveRailTools() {
	if v == nil || v.state == nil {
		return
	}
	root := v.state.Workspace().Root
	if root == "" {
		return
	}
	if v.railStateByWorkspace == nil {
		v.railStateByWorkspace = map[string]railWorkspaceState{}
	}
	v.railStateByWorkspace[root] = railWorkspaceState{
		LeftTool:  firstNonEmptyString(v.activeLeftRailTool, defaultLeftRailTool),
		RightTool: firstNonEmptyString(v.activeRightRailTool, defaultRightRailTool),
	}
}

func (v *View) restoreActiveRailTools(root string) {
	if v == nil {
		return
	}
	state, ok := v.railStateByWorkspace[root]
	if !ok {
		state = railWorkspaceState{LeftTool: defaultLeftRailTool, RightTool: defaultRightRailTool}
	}
	v.activeLeftRailTool = firstNonEmptyString(state.LeftTool, defaultLeftRailTool)
	v.activeRightRailTool = firstNonEmptyString(state.RightTool, defaultRightRailTool)
	v.refreshRailActiveState()
}

func leftRailLabelForTab(title string) string {
	for _, tool := range leftRailToolWindows() {
		if tool.TargetTab == title {
			return tool.Label
		}
	}
	return ""
}

func rightRailLabelForTab(title string) string {
	switch title {
	case "Artifacts":
		return "Sources"
	case "Jobs":
		return "Monitor"
	case "Diagnostics":
		return "Inspector"
	default:
		return ""
	}
}

package shell

import "fyne.io/fyne/v2/widget"

func (v *View) setLeftRailActive(label string) {
	v.activeLeftRailTool = label
	v.refreshRailActiveState()
}

func (v *View) setRightRailActive(label string) {
	v.activeRightRailTool = label
	v.refreshRailActiveState()
}

func (v *View) updateRailActiveStateForTab(title string) {
	if label := leftRailLabelForTab(title); label != "" {
		v.activeLeftRailTool = label
	}
	if label := rightRailLabelForTab(title); label != "" {
		v.activeRightRailTool = label
	}
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

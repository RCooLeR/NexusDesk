package shell

type rightRailToolWindow = toolWindowRegistration

func rightRailToolWindows() []rightRailToolWindow {
	return defaultToolWindowRegistry().ForSide(toolWindowSideRight)
}

func (v *View) openRightRailToolWindow(tool rightRailToolWindow) {
	if tool.FocusAssistant {
		if v.window != nil && v.assistant != nil && v.assistant.prompt != nil {
			v.window.Canvas().Focus(v.assistant.prompt)
		}
		v.setRightRailActive(tool.Label)
		v.publishShellEvent(toolWindowSelectedEvent(tool))
		v.addActivity(tool.Activity)
		return
	}
	if tool.TargetTab == "" {
		v.addActivity(tool.Label + " is unavailable.")
		return
	}
	if !v.selectBottomTab(tool.TargetTab) {
		v.addActivity(tool.Label + " panel is unavailable.")
		return
	}
	v.setRightRailActive(tool.Label)
	v.publishShellEvent(toolWindowSelectedEvent(tool))
	v.addActivity(tool.Activity)
}

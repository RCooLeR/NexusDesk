package shell

type leftRailToolWindow = toolWindowRegistration

func leftRailToolWindows() []leftRailToolWindow {
	return defaultToolWindowRegistry().ForSide(toolWindowSideLeft)
}

func (v *View) openLeftRailToolWindow(tool leftRailToolWindow) {
	if tool.OpenProject {
		v.openHomeTab()
		v.setLeftRailActive(tool.Label)
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
	v.setLeftRailActive(tool.Label)
	v.publishShellEvent(toolWindowSelectedEvent(tool))
	v.addActivity(tool.Activity)
}

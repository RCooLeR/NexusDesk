package shell

func (v *View) addActivity(message string) {
	v.activityText += "\n\n" + message
	v.activityLog.ParseMarkdown(v.activityText)
}

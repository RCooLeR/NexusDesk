package shell

import "strings"

const activityHistoryLimit = 400

func (v *View) addActivity(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	v.appendActivityLine(message)
	v.activityText = strings.Join(v.activityLines, "\n\n")
	v.activityLog.ParseMarkdown(v.activityText)
}

func (v *View) appendActivityLine(message string) {
	v.activityLines = append(v.activityLines, message)
	if len(v.activityLines) > activityHistoryLimit {
		start := len(v.activityLines) - activityHistoryLimit
		tail := make([]string, activityHistoryLimit)
		copy(tail, v.activityLines[start:])
		v.activityLines = tail
	}
}

func (v *View) recentActivityLines(limit int) []string {
	if limit <= 0 || len(v.activityLines) == 0 {
		return nil
	}
	start := 0
	if len(v.activityLines) > limit {
		start = len(v.activityLines) - limit
	}
	tail := make([]string, len(v.activityLines)-start)
	copy(tail, v.activityLines[start:])
	return tail
}

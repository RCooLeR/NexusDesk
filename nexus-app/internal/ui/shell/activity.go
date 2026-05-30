package shell

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	activityHistoryLimit = 400
	activityRenderLimit  = 120
)

func (v *View) addActivity(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	v.appendActivityLine(message)
	v.activityText = activityMarkdown(v.recentActivityLines(activityRenderLimit))
	v.appendActivityRow(message)
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

func activityMarkdown(lines []string) string {
	return strings.Join(lines, "\n\n")
}

func newActivityList(lines []string) *fyne.Container {
	rows := make([]fyne.CanvasObject, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, newActivityRow(line))
	}
	return container.NewVBox(rows...)
}

func (v *View) appendActivityRow(message string) {
	if v.activityList == nil {
		return
	}
	for len(v.activityList.Objects) >= activityRenderLimit {
		v.activityList.Objects = v.activityList.Objects[1:]
	}
	v.activityList.Add(newActivityRow(message))
	v.activityList.Refresh()
}

func newActivityRow(message string) fyne.CanvasObject {
	label := widget.NewLabel(message)
	label.Wrapping = fyne.TextWrapWord
	return label
}

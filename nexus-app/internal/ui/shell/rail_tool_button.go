package shell

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

var _ desktop.Hoverable = (*railToolButton)(nil)

type railToolButton struct {
	widget.Button
	tooltip string
	onHover func(string)
	onLeave func()
}

func newRailIconButton(tool toolWindowRegistration, action func(), onHover func(string), onLeave func()) *railToolButton {
	button := &railToolButton{
		tooltip: railToolTipText(tool),
		onHover: onHover,
		onLeave: onLeave,
	}
	button.Text = ""
	button.Icon = tool.Icon
	button.OnTapped = action
	button.Importance = widget.LowImportance
	button.ExtendBaseWidget(button)
	return button
}

func (b *railToolButton) MouseIn(event *desktop.MouseEvent) {
	b.Button.MouseIn(event)
	if b.onHover != nil && strings.TrimSpace(b.tooltip) != "" {
		b.onHover(b.tooltip)
	}
}

func (b *railToolButton) MouseMoved(event *desktop.MouseEvent) {
	b.Button.MouseMoved(event)
}

func (b *railToolButton) MouseOut() {
	b.Button.MouseOut()
	if b.onLeave != nil {
		b.onLeave()
	}
}

func railToolTipText(tool toolWindowRegistration) string {
	if strings.TrimSpace(tool.Shortcut) == "" {
		return tool.Label
	}
	return fmt.Sprintf("%s (%s)", tool.Label, tool.Shortcut)
}

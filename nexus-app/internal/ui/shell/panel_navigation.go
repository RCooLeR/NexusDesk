package shell

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	uiTheme "nexusdesk/internal/ui/theme"
)

const (
	workbenchExpandedOffset   = 0.24
	minToolPanelOffset        = 0.16
	maxToolPanelOffset        = 0.42
	editorWidthPriorityOffset = 0.82
)

func (v *View) selectBottomTab(title string) bool {
	if v.bottomTabs == nil {
		return false
	}
	if !selectAppTabByTitle(v.bottomTabs, title) {
		return false
	}
	v.enforceEditorWidthPriority()
	v.updateRailActiveStateForTab(title)
	return true
}

func (v *View) collapseBottomPanel() {
	if v == nil {
		return
	}
	v.rememberCurrentToolPanelOffset()
	v.bottomPanelCollapsed = true
	if v.workbenchSplit != nil {
		v.workbenchSplit.SetOffset(0)
	}
}

func (v *View) expandBottomPanel() {
	v.expandToolPanelFor(v.currentToolPanelKey())
}

func (v *View) expandToolPanelFor(label string) {
	if v == nil {
		return
	}
	if label != "" {
		v.activeToolPanelKey = label
	}
	v.bottomPanelCollapsed = false
	if v.workbenchSplit != nil {
		v.workbenchSplit.SetOffset(v.toolPanelOffsetFor(label))
	}
}

func (v *View) newEditorPrioritySplit(rightWorkbench fyne.CanvasObject) *container.Split {
	split := container.NewHSplit(v.editor.tabs, rightWorkbench)
	split.SetOffset(editorWidthPriorityOffset)
	v.mainSplit = split
	return split
}

func (v *View) newToolPanelSplit(workbench fyne.CanvasObject) *container.Split {
	toolPanel := container.NewBorder(nil, nil, nil, newToolPanelResizeHandle(v), v.newBottomPanel())
	split := container.NewHSplit(toolPanel, workbench)
	split.SetOffset(v.toolPanelOffsetFor(v.currentToolPanelKey()))
	v.workbenchSplit = split
	return split
}

func (v *View) rememberCurrentToolPanelOffset() {
	if v == nil || v.workbenchSplit == nil || v.bottomPanelCollapsed {
		return
	}
	key := v.currentToolPanelKey()
	if key == "" {
		return
	}
	offset := v.workbenchSplit.Offset
	if offset < minToolPanelOffset || offset > maxToolPanelOffset {
		return
	}
	if v.toolPanelOffsetByTool == nil {
		v.toolPanelOffsetByTool = map[string]float64{}
	}
	v.toolPanelOffsetByTool[key] = offset
}

func (v *View) toolPanelOffsetFor(label string) float64 {
	if v == nil || label == "" || v.toolPanelOffsetByTool == nil {
		return workbenchExpandedOffset
	}
	offset, ok := v.toolPanelOffsetByTool[label]
	if !ok || offset < minToolPanelOffset || offset > maxToolPanelOffset {
		return workbenchExpandedOffset
	}
	return offset
}

func (v *View) currentToolPanelKey() string {
	if v == nil {
		return ""
	}
	if v.activeToolPanelKey != "" {
		return v.activeToolPanelKey
	}
	if v.activeLeftRailTool != "" {
		return v.activeLeftRailTool
	}
	if v.activeRightRailTool != "" {
		return v.activeRightRailTool
	}
	return defaultLeftRailTool
}

func (v *View) enforceEditorWidthPriority() {
	if v == nil || v.mainSplit == nil {
		return
	}
	if v.mainSplit.Offset < editorWidthPriorityOffset {
		v.mainSplit.SetOffset(editorWidthPriorityOffset)
	}
}

type toolPanelResizeHandle struct {
	widget.BaseWidget
	view *View
}

func newToolPanelResizeHandle(view *View) *toolPanelResizeHandle {
	handle := &toolPanelResizeHandle{view: view}
	handle.ExtendBaseWidget(handle)
	return handle
}

func (h *toolPanelResizeHandle) MinSize() fyne.Size {
	return fyne.NewSize(uiTheme.DensityForMode(uiTheme.DensityCompact).ResizeHandleHitWidth, 1)
}

func (h *toolPanelResizeHandle) Dragged(event *fyne.DragEvent) {
	if h == nil || h.view == nil || h.view.workbenchSplit == nil || event == nil {
		return
	}
	width := h.view.workbenchSplit.Size().Width
	if width <= 0 {
		return
	}
	next := h.view.workbenchSplit.Offset + float64(event.Dragged.DX/width)
	h.view.workbenchSplit.SetOffset(clampToolPanelOffset(next))
	h.view.rememberCurrentToolPanelOffset()
}

func (h *toolPanelResizeHandle) DragEnd() {}

func (h *toolPanelResizeHandle) CreateRenderer() fyne.WidgetRenderer {
	grip := canvas.NewRectangle(uiTheme.JetBrainsDarkPalette().Border)
	return &toolPanelResizeHandleRenderer{handle: h, grip: grip}
}

type toolPanelResizeHandleRenderer struct {
	handle *toolPanelResizeHandle
	grip   *canvas.Rectangle
}

func (r *toolPanelResizeHandleRenderer) Layout(size fyne.Size) {
	width := float32(2)
	r.grip.Resize(fyne.NewSize(width, size.Height))
	r.grip.Move(fyne.NewPos((size.Width-width)/2, 0))
}

func (r *toolPanelResizeHandleRenderer) MinSize() fyne.Size {
	if r.handle == nil {
		return fyne.NewSize(uiTheme.DensityForMode(uiTheme.DensityCompact).ResizeHandleHitWidth, 1)
	}
	return r.handle.MinSize()
}

func (r *toolPanelResizeHandleRenderer) Refresh() {
	r.grip.FillColor = uiTheme.JetBrainsDarkPalette().Border
	r.grip.Refresh()
}

func (r *toolPanelResizeHandleRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.grip}
}

func (r *toolPanelResizeHandleRenderer) Destroy() {}

func clampToolPanelOffset(offset float64) float64 {
	if offset < minToolPanelOffset {
		return minToolPanelOffset
	}
	if offset > maxToolPanelOffset {
		return maxToolPanelOffset
	}
	return offset
}

func selectAppTabByTitle(tabs *container.AppTabs, title string) bool {
	if tabs == nil {
		return false
	}
	for _, item := range tabs.Items {
		if strings.EqualFold(item.Text, title) {
			tabs.Select(item)
			return true
		}
		childTabs, ok := item.Content.(*container.AppTabs)
		if !ok || !selectAppTabByTitle(childTabs, title) {
			continue
		}
		tabs.Select(item)
		return true
	}
	return false
}

func (v *View) isBottomTabSelected(title string) bool {
	if v.bottomTabs == nil {
		return false
	}
	return isAppTabSelected(v.bottomTabs, title)
}

func isAppTabSelected(tabs *container.AppTabs, title string) bool {
	if tabs == nil {
		return false
	}
	selected := tabs.Selected()
	if selected == nil {
		return false
	}
	if strings.EqualFold(selected.Text, title) {
		return true
	}
	childTabs, ok := selected.Content.(*container.AppTabs)
	return ok && isAppTabSelected(childTabs, title)
}

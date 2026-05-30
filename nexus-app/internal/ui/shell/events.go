package shell

import "fyne.io/fyne/v2"

type shellController interface {
	Panel() fyne.CanvasObject
}

type refreshableShellController interface {
	Refresh()
}

type shellEventType string

const (
	shellEventToolWindowSelected shellEventType = "tool_window_selected"
)

type shellEvent struct {
	Type      shellEventType
	ToolID    string
	ToolLabel string
	TabTitle  string
	Message   string
}

type shellEventHandler func(shellEvent)

type shellEventBus struct {
	subscribers map[shellEventType][]shellEventHandler
}

func newShellEventBus() *shellEventBus {
	return &shellEventBus{subscribers: map[shellEventType][]shellEventHandler{}}
}

func (b *shellEventBus) Subscribe(eventType shellEventType, handler shellEventHandler) {
	if b == nil || handler == nil {
		return
	}
	if b.subscribers == nil {
		b.subscribers = map[shellEventType][]shellEventHandler{}
	}
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *shellEventBus) Publish(event shellEvent) {
	if b == nil {
		return
	}
	for _, handler := range append([]shellEventHandler{}, b.subscribers[event.Type]...) {
		handler(event)
	}
}

func (v *View) publishShellEvent(event shellEvent) {
	if v == nil || v.events == nil {
		return
	}
	v.events.Publish(event)
}

package shell

import "testing"

func TestShellEventBusPublishesTypedEvents(t *testing.T) {
	bus := newShellEventBus()
	received := []shellEvent{}
	bus.Subscribe(shellEventToolWindowSelected, func(event shellEvent) {
		received = append(received, event)
	})

	bus.Publish(shellEvent{Type: shellEventToolWindowSelected, ToolID: "search", ToolLabel: "Search"})

	if len(received) != 1 || received[0].ToolID != "search" || received[0].ToolLabel != "Search" {
		t.Fatalf("expected typed tool-window event, got %#v", received)
	}
}

func TestToolWindowSelectedEventCarriesRegistryMetadata(t *testing.T) {
	tool, ok := defaultToolWindowRegistry().Lookup("artifacts")
	if !ok {
		t.Fatal("expected artifacts tool registration")
	}

	event := toolWindowSelectedEvent(tool)

	if event.Type != shellEventToolWindowSelected || event.ToolID != "artifacts" || event.TabTitle != "Artifacts" {
		t.Fatalf("unexpected tool-window event: %#v", event)
	}
}

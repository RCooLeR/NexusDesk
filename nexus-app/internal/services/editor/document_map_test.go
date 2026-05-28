package editor

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildDocumentMapCombinesSymbolsAndMarkers(t *testing.T) {
	content := strings.Join([]string{
		"package main",
		"",
		"type Runner struct{}",
		"",
		"func Start() {",
		"  // TODO: wire startup",
		"}",
		"<<<<<<< HEAD",
		"=======",
		">>>>>>> branch",
	}, "\n")

	items := BuildDocumentMap("main.go", content)
	got := documentMapSignatures(items)
	want := []string{
		"type:Runner@3",
		"func:Start@5",
		"todo:TODO: wire startup@6",
		"conflict:Merge conflict start@8",
		"conflict:Merge conflict separator@9",
		"conflict:Merge conflict end@10",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected document map\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBuildDocumentMapAddsAnchorsForLongUnstructuredFiles(t *testing.T) {
	var builder strings.Builder
	for i := 1; i <= 120; i++ {
		builder.WriteString(fmt.Sprintf("log line %d\n", i))
	}

	items := BuildDocumentMap("server.log", builder.String())

	if len(items) == 0 {
		t.Fatal("expected anchors for long unstructured file")
	}
	if items[0].Kind != "anchor" || items[0].Line <= 0 {
		t.Fatalf("expected first item to be an anchor, got %#v", items[0])
	}
}

func TestBuildDocumentMapCapsLargeFiles(t *testing.T) {
	var builder strings.Builder
	for i := 0; i < documentMapMaxItems+40; i++ {
		builder.WriteString(fmt.Sprintf("func Item%d() {}\n", i))
	}

	items := BuildDocumentMap("main.go", builder.String())

	if len(items) != documentMapMaxItems {
		t.Fatalf("expected %d capped map items, got %d", documentMapMaxItems, len(items))
	}
}

func documentMapSignatures(items []DocumentMapItem) []string {
	signatures := make([]string, 0, len(items))
	for _, item := range items {
		signatures = append(signatures, fmt.Sprintf("%s:%s@%d", item.Kind, item.Label, item.Line))
	}
	return signatures
}

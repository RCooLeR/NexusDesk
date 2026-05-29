package workspace

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConflictMarkersKeepsOurs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "notes.txt"), "before\n<<<<<<< HEAD\nours\n=======\ntheirs\n>>>>>>> branch\nafter\n")

	result, err := New().ResolveConflictMarkers(root, "notes.txt", "ours")
	if err != nil {
		t.Fatalf("ResolveConflictMarkers returned error: %v", err)
	}
	if result.ConflictCount != 1 || result.Strategy != ConflictResolutionOurs || result.Encoding != encodingUTF8 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Content != "before\nours\nafter\n" {
		t.Fatalf("unexpected resolved content: %q", result.Content)
	}
}

func TestResolveConflictMarkersKeepsTheirsAndPreservesCRLF(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "notes.txt"), "before\r\n<<<<<<< HEAD\r\nours\r\n=======\r\ntheirs\r\n>>>>>>> branch\r\nafter\r\n")

	result, err := New().ResolveConflictMarkers(root, "notes.txt", "incoming")
	if err != nil {
		t.Fatalf("ResolveConflictMarkers returned error: %v", err)
	}
	if result.Strategy != ConflictResolutionTheirs {
		t.Fatalf("expected theirs strategy, got %#v", result)
	}
	if result.Content != "before\r\ntheirs\r\nafter\r\n" {
		t.Fatalf("unexpected resolved content: %q", result.Content)
	}
}

func TestResolveConflictMarkersCombinesBothAndIgnoresDiff3Base(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "notes.txt"), "before\n<<<<<<< HEAD\nours\n||||||| base\nbase\n=======\ntheirs\n>>>>>>> branch\nafter\n")

	result, err := New().ResolveConflictMarkers(root, "notes.txt", "both")
	if err != nil {
		t.Fatalf("ResolveConflictMarkers returned error: %v", err)
	}
	if result.Content != "before\nours\ntheirs\nafter\n" {
		t.Fatalf("unexpected resolved content: %q", result.Content)
	}
}

func TestResolveConflictMarkersRejectsInvalidInput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "clean.txt"), "no conflicts\n")
	writeFile(t, filepath.Join(root, "broken.txt"), "<<<<<<< HEAD\nours\n")

	service := New()
	if _, err := service.ResolveConflictMarkers(root, "clean.txt", "ours"); err == nil || !strings.Contains(err.Error(), "no conflict") {
		t.Fatalf("expected no-conflict rejection, got %v", err)
	}
	if _, err := service.ResolveConflictMarkers(root, "broken.txt", "ours"); err == nil || !strings.Contains(err.Error(), "unterminated") {
		t.Fatalf("expected unterminated conflict rejection, got %v", err)
	}
	if _, err := service.ResolveConflictMarkers(root, "clean.txt", "invalid"); err == nil || !strings.Contains(err.Error(), "strategy") {
		t.Fatalf("expected invalid strategy rejection, got %v", err)
	}
	if _, err := service.ResolveConflictMarkers(root, "../outside.txt", "ours"); err == nil {
		t.Fatal("expected traversal target to be rejected")
	}
}

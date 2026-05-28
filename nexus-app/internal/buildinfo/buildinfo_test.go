package buildinfo

import (
	"strings"
	"testing"
)

func TestCurrentBuildInfoDefaultsAreValid(t *testing.T) {
	if err := Current().Validate(); err != nil {
		t.Fatalf("default build info should validate: %v", err)
	}
}

func TestBuildInfoValidateRejectsInvalidVersion(t *testing.T) {
	info := Current()
	info.Version = "dev"
	if err := info.Validate(); err == nil || !strings.Contains(err.Error(), "semantic version") {
		t.Fatalf("expected semantic version validation error, got %v", err)
	}
}

func TestBuildInfoValidateRejectsInvalidBuildDate(t *testing.T) {
	info := Current()
	info.BuildDate = "today"
	if err := info.Validate(); err == nil || !strings.Contains(err.Error(), "RFC3339") {
		t.Fatalf("expected RFC3339 validation error, got %v", err)
	}
}

func TestAboutTextIncludesReleaseIdentity(t *testing.T) {
	text := AboutText()
	for _, expected := range []string{AppName, Tagline, "Version:", "Commit:", "Build:"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected AboutText to contain %q, got %q", expected, text)
		}
	}
}

package userguide

import (
	"strings"
	"testing"
)

func TestAppDataCleanupGuideCoversStorageAndUninstallTopics(t *testing.T) {
	markdown := AppDataCleanupMarkdown()
	for _, expected := range []string{
		"App Data And Uninstall Cleanup",
		"NexusDesk/settings.json",
		"NexusDesk/recent-workspaces.json",
		"NexusDesk/connector-profiles.json",
		"NexusAugenticStudio/assistant-profile.json",
		"DPAPI",
		"Keychain",
		"Secret Service",
		".nexusdesk/",
		"rollbacks",
		"Manual Cleanup",
		"workspace state backup",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected %q in app data cleanup markdown:\n%s", expected, markdown)
		}
	}
}

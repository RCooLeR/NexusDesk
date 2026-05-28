package editor

import (
	"strings"
	"testing"
)

func TestNativeParityEditorStrategyAcceptsSyntaxMirrorForBeta(t *testing.T) {
	strategy := NativeParityEditorStrategy()
	if strategy.Status != "accepted-for-native-parity-beta" || strategy.BetaBlocker {
		t.Fatalf("unexpected strategy status: %#v", strategy)
	}
	for _, expected := range []string{"Syntax mirror", "Document Map", "post-beta"} {
		if !strings.Contains(strategy.Summary(), expected) {
			t.Fatalf("strategy summary missing %q:\n%s", expected, strategy.Summary())
		}
	}
	if len(strategy.NextMilestones) == 0 {
		t.Fatal("expected post-beta editor milestones")
	}
}

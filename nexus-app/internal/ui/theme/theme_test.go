package theme

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	fynetheme "fyne.io/fyne/v2/theme"
)

func TestJetBrainsDarkPaletteDefinesCoreTokens(t *testing.T) {
	palette := JetBrainsDarkPalette()
	assertOpaque(t, "background", palette.Background)
	assertOpaque(t, "panel", palette.Panel)
	assertOpaque(t, "editor", palette.Editor)
	assertOpaque(t, "text primary", palette.TextPrimary)
	assertOpaque(t, "accent", palette.Accent)
	assertOpaque(t, "selection", palette.Selection)
	assertOpaque(t, "success", palette.Success)
	assertOpaque(t, "warning", palette.Warning)
	assertOpaque(t, "error", palette.Error)
	if palette.Background == palette.Panel || palette.Panel == palette.Editor {
		t.Fatalf("expected layered backgrounds, got %#v", palette)
	}
	if palette.Accent == palette.Error || palette.Warning == palette.Success {
		t.Fatalf("expected distinct semantic colors, got %#v", palette)
	}
}

func TestNexusThemeMapsFyneColorsToPalette(t *testing.T) {
	palette := JetBrainsDarkPalette()
	nexusTheme := NexusTheme{}
	cases := []struct {
		name fyne.ThemeColorName
		want color.NRGBA
	}{
		{name: fynetheme.ColorNameBackground, want: palette.Background},
		{name: fynetheme.ColorNameForeground, want: palette.TextPrimary},
		{name: fynetheme.ColorNamePrimary, want: palette.Accent},
		{name: fynetheme.ColorNameForegroundOnPrimary, want: palette.AccentForeground},
		{name: fynetheme.ColorNameInputBorder, want: palette.InputBorder},
		{name: fynetheme.ColorNameFocus, want: palette.Focus},
		{name: fynetheme.ColorNameWarning, want: palette.Warning},
		{name: fynetheme.ColorNameError, want: palette.Error},
		{name: fynetheme.ColorNameSuccess, want: palette.Success},
	}
	for _, tc := range cases {
		if got := nexusTheme.Color(tc.name, fynetheme.VariantDark); got != tc.want {
			t.Fatalf("expected %s to map to %#v, got %#v", tc.name, tc.want, got)
		}
	}
}

func TestNexusThemeDensitySizesStayCompact(t *testing.T) {
	nexusTheme := NexusTheme{}
	if got := nexusTheme.Size(fynetheme.SizeNamePadding); got != 8 {
		t.Fatalf("expected compact padding 8, got %f", got)
	}
	if got := nexusTheme.Size(fynetheme.SizeNameInnerPadding); got != 6 {
		t.Fatalf("expected compact inner padding 6, got %f", got)
	}
}

func assertOpaque(t *testing.T, label string, value color.NRGBA) {
	t.Helper()
	if value.A != 255 {
		t.Fatalf("expected %s token to be opaque, got %#v", label, value)
	}
}

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
	assertOpaque(t, "raised panel", palette.PanelRaised)
	assertOpaque(t, "editor", palette.Editor)
	assertOpaque(t, "status bar", palette.StatusBar)
	assertOpaque(t, "text primary", palette.TextPrimary)
	assertOpaque(t, "accent", palette.Accent)
	assertOpaque(t, "active tab underline", palette.ActiveTabUnderline)
	assertOpaque(t, "selection", palette.Selection)
	assertOpaque(t, "success", palette.Success)
	assertOpaque(t, "warning", palette.Warning)
	assertOpaque(t, "error", palette.Error)
	assertOpaque(t, "syntax keyword", palette.SyntaxKeyword)
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
	if density := DensityForMode(DensityCompact); density.RowHeight != 28 || density.FocusRing != 2 || density.ActiveTabUnderline != 2 {
		t.Fatalf("unexpected compact density tokens: %#v", density)
	}
}

func TestNexusThemeComfortableDensitySizes(t *testing.T) {
	nexusTheme := NexusTheme{Density: DensityComfortable}
	if got := nexusTheme.Size(fynetheme.SizeNamePadding); got != 12 {
		t.Fatalf("expected comfortable padding 12, got %f", got)
	}
	if got := nexusTheme.Size(fynetheme.SizeNameInnerPadding); got != 8 {
		t.Fatalf("expected comfortable inner padding 8, got %f", got)
	}
	if density := DensityForMode(DensityComfortable); density.RowHeight != 34 || density.ResizeHandleHitWidth != 12 {
		t.Fatalf("unexpected comfortable density tokens: %#v", density)
	}
}

func TestDensityForModeFallsBackToCompact(t *testing.T) {
	if got := DensityForMode(DensityMode("unknown")); got != DensityForMode(DensityCompact) {
		t.Fatalf("expected unknown density to fall back to compact, got %#v", got)
	}
}

func TestPaletteDiagnosticsPassForProductionPalette(t *testing.T) {
	if issues := PaletteDiagnostics(JetBrainsDarkPalette()); len(issues) != 0 {
		t.Fatalf("expected production palette to pass diagnostics, got %#v", issues)
	}
}

func TestContrastRatioFlagsLowContrastPairs(t *testing.T) {
	low := ContrastRatio(color.NRGBA{R: 20, G: 20, B: 20, A: 255}, color.NRGBA{R: 21, G: 21, B: 21, A: 255})
	if low >= 3 {
		t.Fatalf("expected near-identical colors to have low contrast, got %f", low)
	}
	high := ContrastRatio(color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	if high < 21 {
		t.Fatalf("expected white on black contrast near 21, got %f", high)
	}
}

func assertOpaque(t *testing.T, label string, value color.NRGBA) {
	t.Helper()
	if value.A != 255 {
		t.Fatalf("expected %s token to be opaque, got %#v", label, value)
	}
}

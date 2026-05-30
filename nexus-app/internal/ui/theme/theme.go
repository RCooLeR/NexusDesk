package theme

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type DensityMode string

const (
	DensityCompact     DensityMode = "compact"
	DensityComfortable DensityMode = "comfortable"
)

type Density struct {
	Padding              float32
	InnerPadding         float32
	RowHeight            float32
	FocusRing            float32
	ActiveTabUnderline   float32
	ResizeHandleHitWidth float32
}

type NexusTheme struct {
	Density DensityMode
}

type Palette struct {
	Background         color.NRGBA
	Panel              color.NRGBA
	PanelRaised        color.NRGBA
	Editor             color.NRGBA
	StatusBar          color.NRGBA
	Border             color.NRGBA
	Shadow             color.NRGBA
	TextPrimary        color.NRGBA
	TextSecondary      color.NRGBA
	TextMuted          color.NRGBA
	Accent             color.NRGBA
	AccentHover        color.NRGBA
	AccentPressed      color.NRGBA
	AccentForeground   color.NRGBA
	Selection          color.NRGBA
	Focus              color.NRGBA
	ActiveTabUnderline color.NRGBA
	Input              color.NRGBA
	InputBorder        color.NRGBA
	Button             color.NRGBA
	ButtonDisabled     color.NRGBA
	Success            color.NRGBA
	SuccessForeground  color.NRGBA
	Warning            color.NRGBA
	WarningForeground  color.NRGBA
	Error              color.NRGBA
	ErrorForeground    color.NRGBA
	SyntaxActiveLine   color.NRGBA
	SyntaxComment      color.NRGBA
	SyntaxKeyword      color.NRGBA
	SyntaxNumber       color.NRGBA
	SyntaxString       color.NRGBA
}

var jetBrainsDarkPalette = Palette{
	Background:         color.NRGBA{R: 24, G: 28, B: 35, A: 255},
	Panel:              color.NRGBA{R: 29, G: 34, B: 42, A: 255},
	PanelRaised:        color.NRGBA{R: 35, G: 41, B: 52, A: 255},
	Editor:             color.NRGBA{R: 18, G: 21, B: 27, A: 255},
	StatusBar:          color.NRGBA{R: 22, G: 26, B: 32, A: 255},
	Border:             color.NRGBA{R: 55, G: 63, B: 76, A: 255},
	Shadow:             color.NRGBA{R: 0, G: 0, B: 0, A: 96},
	TextPrimary:        color.NRGBA{R: 235, G: 238, B: 242, A: 255},
	TextSecondary:      color.NRGBA{R: 180, G: 188, B: 200, A: 255},
	TextMuted:          color.NRGBA{R: 132, G: 142, B: 157, A: 255},
	Accent:             color.NRGBA{R: 69, G: 140, B: 255, A: 255},
	AccentHover:        color.NRGBA{R: 84, G: 154, B: 255, A: 255},
	AccentPressed:      color.NRGBA{R: 44, G: 98, B: 180, A: 255},
	AccentForeground:   color.NRGBA{R: 248, G: 251, B: 255, A: 255},
	Selection:          color.NRGBA{R: 44, G: 85, B: 145, A: 255},
	Focus:              color.NRGBA{R: 91, G: 156, B: 255, A: 220},
	ActiveTabUnderline: color.NRGBA{R: 91, G: 156, B: 255, A: 255},
	Input:              color.NRGBA{R: 31, G: 36, B: 45, A: 255},
	InputBorder:        color.NRGBA{R: 72, G: 82, B: 96, A: 255},
	Button:             color.NRGBA{R: 35, G: 41, B: 52, A: 255},
	ButtonDisabled:     color.NRGBA{R: 50, G: 56, B: 66, A: 180},
	Success:            color.NRGBA{R: 68, G: 170, B: 118, A: 255},
	SuccessForeground:  color.NRGBA{R: 9, G: 31, B: 20, A: 255},
	Warning:            color.NRGBA{R: 230, G: 171, B: 67, A: 255},
	WarningForeground:  color.NRGBA{R: 38, G: 26, B: 6, A: 255},
	Error:              color.NRGBA{R: 235, G: 95, B: 95, A: 255},
	ErrorForeground:    color.NRGBA{R: 42, G: 9, B: 9, A: 255},
	SyntaxActiveLine:   color.NRGBA{R: 34, G: 48, B: 69, A: 255},
	SyntaxComment:      color.NRGBA{R: 137, G: 148, B: 166, A: 255},
	SyntaxKeyword:      color.NRGBA{R: 125, G: 166, B: 255, A: 255},
	SyntaxNumber:       color.NRGBA{R: 235, G: 185, B: 104, A: 255},
	SyntaxString:       color.NRGBA{R: 121, G: 207, B: 155, A: 255},
}

func JetBrainsDarkPalette() Palette {
	return jetBrainsDarkPalette
}

func DensityForMode(mode DensityMode) Density {
	switch mode {
	case DensityComfortable:
		return Density{Padding: 12, InnerPadding: 8, RowHeight: 34, FocusRing: 2, ActiveTabUnderline: 3, ResizeHandleHitWidth: 12}
	default:
		return Density{Padding: 8, InnerPadding: 6, RowHeight: 28, FocusRing: 2, ActiveTabUnderline: 2, ResizeHandleHitWidth: 10}
	}
}

func PaletteDiagnostics(palette Palette) []string {
	issues := []string{}
	for name, token := range map[string]color.NRGBA{
		"background":         palette.Background,
		"panel":              palette.Panel,
		"panelRaised":        palette.PanelRaised,
		"editor":             palette.Editor,
		"statusBar":          palette.StatusBar,
		"textPrimary":        palette.TextPrimary,
		"accent":             palette.Accent,
		"activeTabUnderline": palette.ActiveTabUnderline,
		"success":            palette.Success,
		"warning":            palette.Warning,
		"error":              palette.Error,
		"syntaxKeyword":      palette.SyntaxKeyword,
		"syntaxString":       palette.SyntaxString,
		"syntaxNumber":       palette.SyntaxNumber,
		"syntaxComment":      palette.SyntaxComment,
		"syntaxActiveLine":   palette.SyntaxActiveLine,
	} {
		if token.A != 255 {
			issues = append(issues, name+" must be opaque")
		}
	}
	for _, pair := range []struct {
		name       string
		foreground color.NRGBA
		background color.NRGBA
		min        float64
	}{
		{name: "primary text on background", foreground: palette.TextPrimary, background: palette.Background, min: 4.5},
		{name: "secondary text on panel", foreground: palette.TextSecondary, background: palette.Panel, min: 3.0},
		{name: "muted text on raised panel", foreground: palette.TextMuted, background: palette.PanelRaised, min: 3.0},
		{name: "accent foreground on accent", foreground: palette.AccentForeground, background: palette.Accent, min: 3.0},
		{name: "success foreground on success", foreground: palette.SuccessForeground, background: palette.Success, min: 3.0},
		{name: "warning foreground on warning", foreground: palette.WarningForeground, background: palette.Warning, min: 3.0},
		{name: "error foreground on error", foreground: palette.ErrorForeground, background: palette.Error, min: 3.0},
		{name: "syntax keyword on editor", foreground: palette.SyntaxKeyword, background: palette.Editor, min: 3.0},
		{name: "syntax string on editor", foreground: palette.SyntaxString, background: palette.Editor, min: 3.0},
		{name: "syntax number on editor", foreground: palette.SyntaxNumber, background: palette.Editor, min: 3.0},
		{name: "syntax comment on editor", foreground: palette.SyntaxComment, background: palette.Editor, min: 3.0},
	} {
		if ContrastRatio(pair.foreground, pair.background) < pair.min {
			issues = append(issues, pair.name+" contrast is below target")
		}
	}
	return issues
}

func ContrastRatio(foreground color.NRGBA, background color.NRGBA) float64 {
	light := relativeLuminance(foreground)
	dark := relativeLuminance(background)
	if dark > light {
		light, dark = dark, light
	}
	return (light + 0.05) / (dark + 0.05)
}

func relativeLuminance(value color.NRGBA) float64 {
	channel := func(raw uint8) float64 {
		v := float64(raw) / 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(value.R) + 0.7152*channel(value.G) + 0.0722*channel(value.B)
}

func (NexusTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	palette := JetBrainsDarkPalette()
	switch name {
	case theme.ColorNameBackground:
		return palette.Background
	case theme.ColorNameForeground:
		return palette.TextPrimary
	case theme.ColorNamePrimary:
		return palette.Accent
	case theme.ColorNameForegroundOnPrimary:
		return palette.AccentForeground
	case theme.ColorNameSelection:
		return palette.Selection
	case theme.ColorNameFocus:
		return palette.Focus
	case theme.ColorNameInputBackground:
		return palette.Input
	case theme.ColorNameInputBorder:
		return palette.InputBorder
	case theme.ColorNameButton:
		return palette.Button
	case theme.ColorNameDisabledButton:
		return palette.ButtonDisabled
	case theme.ColorNameDisabled, theme.ColorNamePlaceHolder:
		return palette.TextMuted
	case theme.ColorNameHover:
		return palette.AccentHover
	case theme.ColorNamePressed:
		return palette.AccentPressed
	case theme.ColorNameHeaderBackground, theme.ColorNameMenuBackground:
		return palette.Panel
	case theme.ColorNameOverlayBackground:
		return palette.PanelRaised
	case theme.ColorNameSeparator:
		return palette.Border
	case theme.ColorNameShadow:
		return palette.Shadow
	case theme.ColorNameSuccess:
		return palette.Success
	case theme.ColorNameForegroundOnSuccess:
		return palette.SuccessForeground
	case theme.ColorNameWarning:
		return palette.Warning
	case theme.ColorNameForegroundOnWarning:
		return palette.WarningForeground
	case theme.ColorNameError:
		return palette.Error
	case theme.ColorNameForegroundOnError:
		return palette.ErrorForeground
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (NexusTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (NexusTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (n NexusTheme) Size(name fyne.ThemeSizeName) float32 {
	density := DensityForMode(n.Density)
	switch name {
	case theme.SizeNamePadding:
		return density.Padding
	case theme.SizeNameInnerPadding:
		return density.InnerPadding
	default:
		return theme.DefaultTheme().Size(name)
	}
}

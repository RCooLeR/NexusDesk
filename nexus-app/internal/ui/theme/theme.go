package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type DensityMode string

const (
	DensityCompact     DensityMode = "compact"
	DensityComfortable DensityMode = "comfortable"
)

type Density struct {
	Padding      float32
	InnerPadding float32
}

type NexusTheme struct {
	Density DensityMode
}

type Palette struct {
	Background        color.NRGBA
	Panel             color.NRGBA
	PanelRaised       color.NRGBA
	Editor            color.NRGBA
	Border            color.NRGBA
	Shadow            color.NRGBA
	TextPrimary       color.NRGBA
	TextSecondary     color.NRGBA
	TextMuted         color.NRGBA
	Accent            color.NRGBA
	AccentHover       color.NRGBA
	AccentPressed     color.NRGBA
	AccentForeground  color.NRGBA
	Selection         color.NRGBA
	Focus             color.NRGBA
	Input             color.NRGBA
	InputBorder       color.NRGBA
	Button            color.NRGBA
	ButtonDisabled    color.NRGBA
	Success           color.NRGBA
	SuccessForeground color.NRGBA
	Warning           color.NRGBA
	WarningForeground color.NRGBA
	Error             color.NRGBA
	ErrorForeground   color.NRGBA
}

var jetBrainsDarkPalette = Palette{
	Background:        color.NRGBA{R: 24, G: 28, B: 35, A: 255},
	Panel:             color.NRGBA{R: 29, G: 34, B: 42, A: 255},
	PanelRaised:       color.NRGBA{R: 35, G: 41, B: 52, A: 255},
	Editor:            color.NRGBA{R: 18, G: 21, B: 27, A: 255},
	Border:            color.NRGBA{R: 55, G: 63, B: 76, A: 255},
	Shadow:            color.NRGBA{R: 0, G: 0, B: 0, A: 96},
	TextPrimary:       color.NRGBA{R: 235, G: 238, B: 242, A: 255},
	TextSecondary:     color.NRGBA{R: 180, G: 188, B: 200, A: 255},
	TextMuted:         color.NRGBA{R: 132, G: 142, B: 157, A: 255},
	Accent:            color.NRGBA{R: 69, G: 140, B: 255, A: 255},
	AccentHover:       color.NRGBA{R: 84, G: 154, B: 255, A: 255},
	AccentPressed:     color.NRGBA{R: 44, G: 98, B: 180, A: 255},
	AccentForeground:  color.NRGBA{R: 248, G: 251, B: 255, A: 255},
	Selection:         color.NRGBA{R: 44, G: 85, B: 145, A: 255},
	Focus:             color.NRGBA{R: 91, G: 156, B: 255, A: 220},
	Input:             color.NRGBA{R: 31, G: 36, B: 45, A: 255},
	InputBorder:       color.NRGBA{R: 72, G: 82, B: 96, A: 255},
	Button:            color.NRGBA{R: 35, G: 41, B: 52, A: 255},
	ButtonDisabled:    color.NRGBA{R: 50, G: 56, B: 66, A: 180},
	Success:           color.NRGBA{R: 68, G: 170, B: 118, A: 255},
	SuccessForeground: color.NRGBA{R: 9, G: 31, B: 20, A: 255},
	Warning:           color.NRGBA{R: 230, G: 171, B: 67, A: 255},
	WarningForeground: color.NRGBA{R: 38, G: 26, B: 6, A: 255},
	Error:             color.NRGBA{R: 235, G: 95, B: 95, A: 255},
	ErrorForeground:   color.NRGBA{R: 42, G: 9, B: 9, A: 255},
}

func JetBrainsDarkPalette() Palette {
	return jetBrainsDarkPalette
}

func DensityForMode(mode DensityMode) Density {
	switch mode {
	case DensityComfortable:
		return Density{Padding: 12, InnerPadding: 8}
	default:
		return Density{Padding: 8, InnerPadding: 6}
	}
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

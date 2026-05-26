package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type NexusTheme struct{}

func (NexusTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 24, G: 28, B: 35, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 235, G: 238, B: 242, A: 255}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 69, G: 140, B: 255, A: 255}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 44, G: 85, B: 145, A: 255}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 31, G: 36, B: 45, A: 255}
	case theme.ColorNameButton:
		return color.NRGBA{R: 35, G: 41, B: 52, A: 255}
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

func (NexusTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInnerPadding:
		return 6
	default:
		return theme.DefaultTheme().Size(name)
	}
}

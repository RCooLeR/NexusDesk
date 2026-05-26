package brand

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/nexus-app-icon-transparent.png
var appIconBytes []byte

//go:embed assets/nexus-horizontal-white.png
var horizontalLogoBytes []byte

func AppIcon() fyne.Resource {
	return fyne.NewStaticResource("nexus-app-icon-transparent.png", appIconBytes)
}

func HorizontalLogo() fyne.Resource {
	return fyne.NewStaticResource("nexus-horizontal-white.png", horizontalLogoBytes)
}

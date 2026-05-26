package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
)

func newFilePreview(preview domain.FilePreview) fyne.CanvasObject {
	header := widget.NewLabel(previewHeader(preview))
	header.TextStyle = fyne.TextStyle{Monospace: true}
	if preview.Kind == domain.PreviewImage {
		return container.NewBorder(header, nil, nil, nil, newImagePreview(preview))
	}
	if preview.Kind != domain.PreviewText {
		return container.NewBorder(header, nil, nil, nil, widget.NewLabel("Binary preview is not available in the first native slice."))
	}
	content := widget.NewMultiLineEntry()
	content.SetText(preview.Text)
	content.Wrapping = fyne.TextWrapOff
	content.Disable()
	return container.NewBorder(header, nil, nil, nil, content)
}

func newImagePreview(preview domain.FilePreview) fyne.CanvasObject {
	resource := fyne.NewStaticResource(preview.Name, preview.Bytes)
	image := canvas.NewImageFromResource(resource)
	image.FillMode = canvas.ImageFillContain
	image.SetMinSize(fyne.NewSize(360, 260))
	return container.NewCenter(image)
}

func previewHeader(preview domain.FilePreview) string {
	if preview.Encoding == "" {
		return fmt.Sprintf("%s - %d bytes - %s", preview.RelPath, preview.Size, preview.MediaType)
	}
	return fmt.Sprintf("%s - %d bytes - %s - %s", preview.RelPath, preview.Size, preview.MediaType, preview.Encoding)
}

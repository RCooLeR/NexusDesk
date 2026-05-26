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
	if preview.Kind == domain.PreviewTable {
		return container.NewBorder(header, nil, nil, nil, newTablePreview(preview))
	}
	if preview.Kind == domain.PreviewDoc {
		return container.NewBorder(header, nil, nil, nil, newDocumentPreview(preview))
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

func newTablePreview(preview domain.FilePreview) fyne.CanvasObject {
	if preview.Table == nil {
		return widget.NewLabel("Table preview is unavailable.")
	}
	headers := preview.Table.Headers
	rows := preview.Table.Rows
	status := widget.NewLabel(tablePreviewStatus(preview))
	table := widget.NewTable(
		func() (int, int) { return len(rows) + 1, maxInt(1, len(headers)) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, object fyne.CanvasObject) {
			label := object.(*widget.Label)
			label.TextStyle = fyne.TextStyle{Monospace: true}
			if id.Row == 0 {
				label.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
				label.SetText(tableCell(headers, id.Col))
				return
			}
			label.SetText(tableCell(rows[id.Row-1], id.Col))
		},
	)
	return container.NewBorder(status, nil, nil, nil, table)
}

func newDocumentPreview(preview domain.FilePreview) fyne.CanvasObject {
	if preview.Document == nil {
		return widget.NewLabel("Document preview is unavailable.")
	}
	status := widget.NewLabel(documentPreviewStatus(preview))
	content := widget.NewMultiLineEntry()
	content.SetText(preview.Document.Text)
	content.Wrapping = fyne.TextWrapWord
	content.Disable()
	return container.NewBorder(status, nil, nil, nil, content)
}

func documentPreviewStatus(preview domain.FilePreview) string {
	if preview.Document.Truncated {
		return "Showing extracted text preview. Document preview is capped."
	}
	return "Showing extracted document text."
}

func tablePreviewStatus(preview domain.FilePreview) string {
	if preview.Table.Truncated {
		return fmt.Sprintf("Showing first %d rows. Table preview is capped.", len(preview.Table.Rows))
	}
	return fmt.Sprintf("Showing %d rows.", len(preview.Table.Rows))
}

func tableCell(values []string, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	return values[index]
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func previewHeader(preview domain.FilePreview) string {
	if preview.Encoding == "" {
		return fmt.Sprintf("%s - %d bytes - %s", preview.RelPath, preview.Size, preview.MediaType)
	}
	return fmt.Sprintf("%s - %d bytes - %s - %s", preview.RelPath, preview.Size, preview.MediaType, preview.Encoding)
}

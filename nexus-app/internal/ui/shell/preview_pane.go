package shell

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
)

type previewPane struct {
	markdown bool
	text     *widget.Entry
	rich     *widget.RichText
}

func newPreviewPane(preview domain.FilePreview, text string) *previewPane {
	pane := &previewPane{markdown: isMarkdownPreview(preview)}
	if pane.markdown {
		pane.rich = widget.NewRichTextFromMarkdown(text)
		pane.rich.Wrapping = fyne.TextWrapWord
		return pane
	}
	pane.text = widget.NewMultiLineEntry()
	pane.text.Wrapping = fyne.TextWrapOff
	pane.text.Disable()
	pane.SetText(text)
	return pane
}

func (p *previewPane) Canvas() fyne.CanvasObject {
	if p.markdown {
		return p.rich
	}
	return p.text
}

func (p *previewPane) SetText(text string) {
	if p.markdown {
		p.rich.ParseMarkdown(text)
		return
	}
	p.text.SetText(text)
}

func isMarkdownPreview(preview domain.FilePreview) bool {
	mediaType := strings.ToLower(preview.MediaType)
	if mediaType == "text/markdown" || mediaType == "text/x-markdown" {
		return true
	}
	ext := strings.ToLower(filepath.Ext(preview.RelPath))
	return ext == ".md" || ext == ".markdown" || ext == ".mdown" || ext == ".mkd"
}

package documents

import "nexusdesk/internal/domain"

type Previewer interface {
	PreviewFile(root string, relPath string) (domain.FilePreview, error)
}

type ExtractedDocument struct {
	RelPath   string
	Title     string
	Format    string
	MediaType string
	Encoding  string
	Text      string
	Size      int64
	Lines     int
	Words     int
	Truncated bool
}

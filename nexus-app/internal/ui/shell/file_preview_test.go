package shell

import (
	"strings"
	"testing"

	"nexusdesk/internal/domain"
)

func TestPreviewHeaderSurfacesEncodingWarning(t *testing.T) {
	header := previewHeader(domain.FilePreview{
		RelPath:         "notes.txt",
		Size:            4,
		MediaType:       "text/plain",
		Encoding:        "iso-8859-1",
		EncodingWarning: "Low-confidence single-byte charset detection.",
	})
	for _, expected := range []string{"notes.txt", "iso-8859-1", "Low-confidence"} {
		if !strings.Contains(header, expected) {
			t.Fatalf("preview header missing %q: %s", expected, header)
		}
	}
}

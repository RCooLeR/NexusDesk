package workspace

import (
	"errors"
	"os"
)

type TextFileRead struct {
	RelPath           string
	Content           string
	Encoding          string
	EncodingWarning   string
	EncodingAmbiguous bool
	Size              int64
}

func (s *Service) ReadTextFile(root string, relPath string) (TextFileRead, error) {
	absTarget, cleanRelPath, info, err := resolveFile(root, relPath)
	if err != nil {
		return TextFileRead{}, err
	}
	if info.Size() > writeContentMaxBytes {
		return TextFileRead{}, errors.New("file is too large for safe text formatting")
	}
	content, err := os.ReadFile(absTarget)
	if err != nil {
		return TextFileRead{}, err
	}
	if looksBinary(content) && !looksLikeUTF16LE(content) && !looksLikeUTF16BE(content) {
		return TextFileRead{}, errors.New("file is not safe text")
	}
	detection, err := decodeTextWithDetection(content)
	if err != nil {
		return TextFileRead{}, err
	}
	return TextFileRead{
		RelPath:           cleanRelPath,
		Content:           detection.Text,
		Encoding:          detection.Encoding,
		EncodingWarning:   detection.Warning,
		EncodingAmbiguous: detection.Ambiguous,
		Size:              info.Size(),
	}, nil
}

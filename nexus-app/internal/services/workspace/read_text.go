package workspace

import (
	"errors"
	"os"
)

type TextFileRead struct {
	RelPath  string
	Content  string
	Encoding string
	Size     int64
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
	text, encoding, err := decodeText(content)
	if err != nil {
		return TextFileRead{}, err
	}
	return TextFileRead{
		RelPath:  cleanRelPath,
		Content:  text,
		Encoding: encoding,
		Size:     info.Size(),
	}, nil
}

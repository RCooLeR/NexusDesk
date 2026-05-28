package workspace

import (
	"bytes"
	"encoding/binary"
	"errors"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

const (
	encodingUTF8        = "utf-8"
	encodingUTF8BOM     = "utf-8-bom"
	encodingUTF16LE     = "utf-16-le"
	encodingUTF16BE     = "utf-16-be"
	encodingWindows1251 = "windows-1251"
	encodingWindows1252 = "windows-1252"
)

func decodeText(content []byte) (string, string, error) {
	switch {
	case bytes.HasPrefix(content, []byte{0xef, 0xbb, 0xbf}):
		return string(content[3:]), encodingUTF8BOM, nil
	case bytes.HasPrefix(content, []byte{0xff, 0xfe}):
		text, err := decodeUTF16(content[2:], binary.LittleEndian)
		return text, encodingUTF16LE, err
	case bytes.HasPrefix(content, []byte{0xfe, 0xff}):
		text, err := decodeUTF16(content[2:], binary.BigEndian)
		return text, encodingUTF16BE, err
	case utf8.Valid(content):
		return string(content), encodingUTF8, nil
	case looksLikeUTF16LE(content):
		text, err := decodeUTF16(content, binary.LittleEndian)
		return text, encodingUTF16LE, err
	case looksLikeUTF16BE(content):
		text, err := decodeUTF16(content, binary.BigEndian)
		return text, encodingUTF16BE, err
	default:
		return decodeLegacyText(content)
	}
}

func decodeLegacyText(content []byte) (string, string, error) {
	type legacyCandidate struct {
		name string
		text string
	}
	candidates := []legacyCandidate{}
	for _, candidate := range []struct {
		name    string
		decoder *charmap.Charmap
	}{
		{name: encodingWindows1251, decoder: charmap.Windows1251},
		{name: encodingWindows1252, decoder: charmap.Windows1252},
	} {
		decoded, err := candidate.decoder.NewDecoder().Bytes(content)
		if err != nil {
			continue
		}
		candidates = append(candidates, legacyCandidate{name: candidate.name, text: string(decoded)})
	}
	if len(candidates) == 0 {
		return "", "", errors.New("file text encoding is unsupported")
	}
	best := candidates[0]
	bestScore := legacyTextScore(best.text, best.name)
	for _, candidate := range candidates[1:] {
		score := legacyTextScore(candidate.text, candidate.name)
		if score > bestScore {
			best = candidate
			bestScore = score
		}
	}
	return best.text, best.name, nil
}

func legacyTextScore(text string, encoding string) int {
	score := 0
	cyrillic := 0
	latin := 0
	for _, r := range text {
		switch {
		case r == '\uFFFD':
			score -= 20
		case unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t':
			score -= 10
		case unicode.In(r, unicode.Cyrillic):
			cyrillic++
			score += 4
		case unicode.In(r, unicode.Latin):
			latin++
			score += 2
		case unicode.IsPrint(r):
			score++
		}
	}
	if cyrillic >= 2 && encoding == encodingWindows1251 {
		score += cyrillic * 4
	}
	if cyrillic == 0 && latin > 0 && encoding == encodingWindows1252 {
		score += latin
	}
	return score
}

func decodeUTF16(content []byte, order binary.ByteOrder) (string, error) {
	if len(content)%2 != 0 {
		return "", errors.New("invalid UTF-16 byte length")
	}
	values := make([]uint16, 0, len(content)/2)
	for index := 0; index < len(content); index += 2 {
		values = append(values, order.Uint16(content[index:index+2]))
	}
	return string(utf16.Decode(values)), nil
}

func looksLikeUTF16LE(content []byte) bool {
	return hasNullPattern(content, 1)
}

func looksLikeUTF16BE(content []byte) bool {
	return hasNullPattern(content, 0)
}

func hasNullPattern(content []byte, offset int) bool {
	if len(content) < 4 {
		return false
	}
	pairs := len(content) / 2
	matches := 0
	for index := 0; index < pairs; index++ {
		if content[index*2+offset] == 0 {
			matches++
		}
	}
	return matches*100/pairs >= 60
}

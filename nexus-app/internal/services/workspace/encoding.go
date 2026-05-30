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
	encodingLatin1      = "iso-8859-1"
	encodingWindows1251 = "windows-1251"
	encodingWindows1252 = "windows-1252"
)

type textEncodingDetection struct {
	Text             string
	Encoding         string
	Warning          string
	Ambiguous        bool
	LosslessFallback bool
}

type legacyCandidate struct {
	name  string
	text  string
	score int
}

func decodeText(content []byte) (string, string, error) {
	detection, err := detectTextEncoding(content)
	if err != nil {
		return "", "", err
	}
	return detection.Text, detection.Encoding, nil
}

func decodeTextWithDetection(content []byte) (textEncodingDetection, error) {
	return detectTextEncoding(content)
}

func detectTextEncoding(content []byte) (textEncodingDetection, error) {
	switch {
	case bytes.HasPrefix(content, []byte{0xef, 0xbb, 0xbf}):
		return textEncodingDetection{Text: string(content[3:]), Encoding: encodingUTF8BOM}, nil
	case bytes.HasPrefix(content, []byte{0xff, 0xfe}):
		text, err := decodeUTF16(content[2:], binary.LittleEndian)
		return textEncodingDetection{Text: text, Encoding: encodingUTF16LE}, err
	case bytes.HasPrefix(content, []byte{0xfe, 0xff}):
		text, err := decodeUTF16(content[2:], binary.BigEndian)
		return textEncodingDetection{Text: text, Encoding: encodingUTF16BE}, err
	case utf8.Valid(content):
		return textEncodingDetection{Text: string(content), Encoding: encodingUTF8}, nil
	case looksLikeUTF16LE(content):
		text, err := decodeUTF16(content, binary.LittleEndian)
		return textEncodingDetection{Text: text, Encoding: encodingUTF16LE}, err
	case looksLikeUTF16BE(content):
		text, err := decodeUTF16(content, binary.BigEndian)
		return textEncodingDetection{Text: text, Encoding: encodingUTF16BE}, err
	default:
		return decodeLegacyText(content)
	}
}

func decodeLegacyText(content []byte) (textEncodingDetection, error) {
	candidates := []legacyCandidate{}
	for _, candidate := range []struct {
		name    string
		decoder *charmap.Charmap
	}{
		{name: encodingWindows1251, decoder: charmap.Windows1251},
		{name: encodingLatin1, decoder: charmap.ISO8859_1},
		{name: encodingWindows1252, decoder: charmap.Windows1252},
	} {
		decoded, err := candidate.decoder.NewDecoder().Bytes(content)
		if err != nil {
			continue
		}
		text := string(decoded)
		candidates = append(candidates, legacyCandidate{name: candidate.name, text: text, score: legacyTextScore(text, candidate.name)})
	}
	if len(candidates) == 0 {
		return textEncodingDetection{}, errors.New("file text encoding is unsupported")
	}
	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.score > best.score {
			best = candidate
		}
	}
	secondScore := best.score
	for _, candidate := range candidates {
		if candidate.name == best.name {
			continue
		}
		if secondScore == best.score || candidate.score > secondScore {
			secondScore = candidate.score
		}
	}
	if best.score-secondScore < 4 {
		fallback := losslessSingleByteFallback(candidates)
		return textEncodingDetection{
			Text:             fallback.text,
			Encoding:         fallback.name,
			Warning:          "Low-confidence single-byte charset detection. NexusDesk used a lossless fallback; choose an explicit save encoding before saving changes.",
			Ambiguous:        true,
			LosslessFallback: true,
		}, nil
	}
	return textEncodingDetection{Text: best.text, Encoding: best.name}, nil
}

func losslessSingleByteFallback(candidates []legacyCandidate) legacyCandidate {
	for _, preferred := range []string{encodingLatin1, encodingWindows1252, encodingWindows1251} {
		for _, candidate := range candidates {
			if candidate.name == preferred {
				return candidate
			}
		}
	}
	return candidates[0]
}

func legacyTextScore(text string, encoding string) int {
	score := 0
	cyrillic := 0
	latin := 0
	for _, r := range text {
		switch {
		case r == '\uFFFD':
			score -= 20
		case r >= '\u0080' && r <= '\u009f':
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
	if cyrillic == 0 && latin > 0 && encoding == encodingLatin1 {
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

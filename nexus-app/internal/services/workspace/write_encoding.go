package workspace

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"

	"golang.org/x/text/encoding/charmap"
)

func encodeWriteContent(content string, requestedEncoding string) ([]byte, string, error) {
	encoding := normalizeWriteEncoding(requestedEncoding)
	return encodeContent(content, encoding, true, requestedEncoding)
}

func encodeAppendContent(content string, requestedEncoding string, existingEncoding string, hasExistingContent bool) ([]byte, string, error) {
	encoding := normalizeWriteEncoding(requestedEncoding)
	if strings.TrimSpace(requestedEncoding) == "" && existingEncoding != "" {
		encoding = existingEncoding
	}
	if hasExistingContent && existingEncoding != "" && encoding != existingEncoding {
		return nil, "", fmt.Errorf("append encoding %q does not match existing file encoding %q", encoding, existingEncoding)
	}
	return encodeContent(content, encoding, !hasExistingContent, requestedEncoding)
}

func encodeContent(content string, encoding string, includeSignature bool, requestedEncoding string) ([]byte, string, error) {
	switch encoding {
	case encodingUTF8:
		return []byte(content), encoding, nil
	case encodingUTF8BOM:
		if includeSignature {
			return append([]byte{0xef, 0xbb, 0xbf}, []byte(content)...), encoding, nil
		}
		return []byte(content), encoding, nil
	case encodingUTF16LE:
		return encodeUTF16(content, binary.LittleEndian, signatureBytes(includeSignature, []byte{0xff, 0xfe})), encoding, nil
	case encodingUTF16BE:
		return encodeUTF16(content, binary.BigEndian, signatureBytes(includeSignature, []byte{0xfe, 0xff})), encoding, nil
	case encodingLatin1:
		encoded, err := charmap.ISO8859_1.NewEncoder().Bytes([]byte(content))
		if err != nil {
			return nil, "", errors.New("content cannot be encoded as iso-8859-1")
		}
		return encoded, encoding, nil
	case encodingWindows1251:
		encoded, err := charmap.Windows1251.NewEncoder().Bytes([]byte(content))
		if err != nil {
			return nil, "", errors.New("content cannot be encoded as windows-1251")
		}
		return encoded, encoding, nil
	case encodingWindows1252:
		encoded, err := charmap.Windows1252.NewEncoder().Bytes([]byte(content))
		if err != nil {
			return nil, "", errors.New("content cannot be encoded as windows-1252")
		}
		return encoded, encoding, nil
	default:
		return nil, "", fmt.Errorf("unsupported write encoding %q", requestedEncoding)
	}
}

func signatureBytes(include bool, signature []byte) []byte {
	if include {
		return signature
	}
	return nil
}

func normalizeWriteEncoding(value string) string {
	encoding := strings.ToLower(strings.TrimSpace(value))
	switch encoding {
	case "", "utf8", encodingUTF8:
		return encodingUTF8
	case "utf8-bom", encodingUTF8BOM, "utf-8 bom":
		return encodingUTF8BOM
	case "utf16le", "utf-16le", "utf-16 le", encodingUTF16LE:
		return encodingUTF16LE
	case "utf16be", "utf-16be", "utf-16 be", encodingUTF16BE:
		return encodingUTF16BE
	case "latin1", "latin-1", "iso8859-1", "iso-8859-1":
		return encodingLatin1
	case "cp1251", "windows1251", encodingWindows1251:
		return encodingWindows1251
	case "cp1252", "windows1252", encodingWindows1252:
		return encodingWindows1252
	default:
		return encoding
	}
}

func encodeUTF16(content string, byteOrder binary.ByteOrder, bom []byte) []byte {
	values := utf16.Encode([]rune(content))
	encoded := make([]byte, 0, len(bom)+len(values)*2)
	encoded = append(encoded, bom...)
	buffer := make([]byte, 2)
	for _, value := range values {
		byteOrder.PutUint16(buffer, value)
		encoded = append(encoded, buffer...)
	}
	return encoded
}

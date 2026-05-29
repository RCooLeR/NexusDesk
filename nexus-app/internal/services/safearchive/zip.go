package safearchive

import (
	"archive/zip"
	"fmt"
	"io"
)

type ZipLimits struct {
	MaxFiles                   int
	MaxMemberUncompressedBytes uint64
	MaxTotalUncompressedBytes  uint64
}

func ValidateZipFiles(files []*zip.File, limits ZipLimits) error {
	if limits.MaxFiles > 0 && len(files) > limits.MaxFiles {
		return fmt.Errorf("zip archive contains %d files, exceeding safety cap of %d", len(files), limits.MaxFiles)
	}

	var total uint64
	for _, file := range files {
		if limits.MaxMemberUncompressedBytes > 0 && file.UncompressedSize64 > limits.MaxMemberUncompressedBytes {
			return fmt.Errorf("zip member %q declares %d uncompressed bytes, exceeding safety cap of %d", file.Name, file.UncompressedSize64, limits.MaxMemberUncompressedBytes)
		}
		if limits.MaxTotalUncompressedBytes > 0 {
			if file.UncompressedSize64 > limits.MaxTotalUncompressedBytes-total {
				return fmt.Errorf("zip archive declares more than safety cap of %d uncompressed bytes", limits.MaxTotalUncompressedBytes)
			}
			total += file.UncompressedSize64
		}
	}
	return nil
}

func ReadZipFile(file *zip.File, maxUncompressedBytes uint64) ([]byte, error) {
	if file == nil {
		return nil, fmt.Errorf("zip member is missing")
	}
	if maxUncompressedBytes > 0 && file.UncompressedSize64 > maxUncompressedBytes {
		return nil, fmt.Errorf("zip member %q declares %d uncompressed bytes, exceeding safety cap of %d", file.Name, file.UncompressedSize64, maxUncompressedBytes)
	}

	body, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return ReadAllLimited(file.Name, body, maxUncompressedBytes)
}

func ReadAllLimited(name string, reader io.Reader, maxBytes uint64) ([]byte, error) {
	if maxBytes == 0 {
		return io.ReadAll(reader)
	}

	limited := io.LimitReader(reader, int64(maxBytes)+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if uint64(len(body)) > maxBytes {
		return nil, fmt.Errorf("zip member %q exceeded safety cap of %d bytes while reading", name, maxBytes)
	}
	return body, nil
}

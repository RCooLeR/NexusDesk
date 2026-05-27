package datasets

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

const parquetMagic = "PAR1"

func profileParquet(root string, relPath string) (Profile, error) {
	path, cleanRelPath, info, err := resolveDatasetFile(root, relPath)
	if err != nil {
		return Profile{}, err
	}
	if info.Size() < 12 {
		return Profile{}, errors.New("parquet file is too small to contain metadata footer")
	}
	file, err := os.Open(path)
	if err != nil {
		return Profile{}, err
	}
	defer file.Close()
	head := make([]byte, 4)
	if _, err := file.ReadAt(head, 0); err != nil {
		return Profile{}, err
	}
	tail := make([]byte, 8)
	if _, err := file.ReadAt(tail, info.Size()-8); err != nil {
		return Profile{}, err
	}
	if string(head) != parquetMagic || string(tail[4:]) != parquetMagic {
		return Profile{}, errors.New("parquet file does not contain PAR1 header/footer magic")
	}
	footerLength := int64(binary.LittleEndian.Uint32(tail[:4]))
	notes := []string{
		"Parquet magic header/footer validated.",
		fmt.Sprintf("Footer metadata length: %d bytes.", footerLength),
		"Column schema and row-group details require the future Parquet reader dependency.",
	}
	return Profile{
		RelPath:   cleanRelPath,
		Format:    "PARQUET",
		MediaType: "application/vnd.apache.parquet",
		Size:      info.Size(),
		Rows:      0,
		Notes:     notes,
	}, nil
}

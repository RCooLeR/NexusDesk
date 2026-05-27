package datasets

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const parquetMagic = "PAR1"
const parquetFooterReadLimit = 4 * 1024 * 1024

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
	if footerLength < 0 || footerLength > info.Size()-12 {
		return Profile{}, fmt.Errorf("parquet footer metadata length %d exceeds file bounds", footerLength)
	}
	dataBytes := info.Size() - footerLength - 8
	notes := []string{
		"Parquet magic header/footer validated.",
		fmt.Sprintf("Footer metadata length: %d bytes.", footerLength),
	}
	parquetProfile := &ParquetProfile{
		FooterLength: footerLength,
		DataBytes:    dataBytes,
	}
	rows := 0
	columns := []ColumnProfile{}
	if footerLength > parquetFooterReadLimit {
		parquetProfile.Truncated = true
		notes = append(notes, fmt.Sprintf("Footer metadata exceeds the %d byte native decode cap; schema and row groups were not decoded.", parquetFooterReadLimit))
	} else if footerLength > 0 {
		footer := make([]byte, footerLength)
		if _, err := file.ReadAt(footer, dataBytes); err != nil && !errors.Is(err, io.EOF) {
			return Profile{}, err
		}
		metadata, err := parseParquetFileMetadata(footer)
		if err != nil {
			notes = append(notes, "Native Parquet footer decode failed: "+err.Error())
		} else {
			parquetProfile.Version = metadata.Version
			parquetProfile.CreatedBy = metadata.CreatedBy
			parquetProfile.SchemaColumns = metadata.SchemaColumns
			parquetProfile.RowGroups = metadata.RowGroups
			parquetProfile.MetadataDecoded = true
			rows = int(clampInt64(metadata.NumRows))
			columns = parquetColumnProfiles(metadata.SchemaColumns, metadata.NumRows)
			notes = append(notes,
				fmt.Sprintf("Decoded Parquet footer metadata: %d schema column(s), %d row group(s).", len(metadata.SchemaColumns), len(metadata.RowGroups)),
				"Dependency decision: native bounded footer decoding is used for schema and row-group summaries; full value reads remain deferred.",
			)
		}
	}
	return Profile{
		RelPath:   cleanRelPath,
		Format:    "PARQUET",
		MediaType: "application/vnd.apache.parquet",
		Size:      info.Size(),
		Rows:      rows,
		Columns:   columns,
		Notes:     notes,
		Parquet:   parquetProfile,
	}, nil
}

func parquetColumnProfiles(columns []ParquetColumn, rows int64) []ColumnProfile {
	profiles := make([]ColumnProfile, 0, len(columns))
	for _, column := range columns {
		name := strings.TrimSpace(column.Path)
		if name == "" {
			name = "column"
		}
		profiles = append(profiles, ColumnProfile{
			Name:     name,
			Type:     parquetColumnType(column),
			NonEmpty: int(clampInt64(rows)),
			Empty:    0,
		})
	}
	return profiles
}

func parquetColumnType(column ParquetColumn) string {
	parts := []string{}
	if strings.TrimSpace(column.Type) != "" {
		parts = append(parts, strings.ToLower(column.Type))
	}
	if strings.TrimSpace(column.ConvertedType) != "" {
		parts = append(parts, strings.ToLower(column.ConvertedType))
	}
	if len(parts) == 0 {
		return "parquet"
	}
	return strings.Join(parts, "/")
}

func clampInt64(value int64) int64 {
	if value < 0 {
		return 0
	}
	if value > int64(^uint(0)>>1) {
		return int64(^uint(0) >> 1)
	}
	return value
}

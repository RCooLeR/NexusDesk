package datasets

import (
	"fmt"
	"strings"
)

const (
	compactStop   = 0
	compactTrue   = 1
	compactFalse  = 2
	compactByte   = 3
	compactI16    = 4
	compactI32    = 5
	compactI64    = 6
	compactDouble = 7
	compactBinary = 8
	compactList   = 9
	compactSet    = 10
	compactMap    = 11
	compactStruct = 12
)

type parquetFileMetadata struct {
	Version       int
	CreatedBy     string
	NumRows       int64
	SchemaColumns []ParquetColumn
	RowGroups     []ParquetRowGroup
}

type parquetSchemaElement struct {
	Name           string
	Type           string
	RepetitionType string
	ConvertedType  string
	NumChildren    int
	TypeLength     int
	Precision      int
	Scale          int
}

func parseParquetFileMetadata(data []byte) (parquetFileMetadata, error) {
	reader := newCompactReader(data)
	metadata := parquetFileMetadata{}
	for {
		field, err := reader.readFieldHeader()
		if err != nil {
			return metadata, err
		}
		if field.Type == compactStop {
			break
		}
		switch field.ID {
		case 1:
			value, err := reader.readI32()
			if err != nil {
				return metadata, err
			}
			metadata.Version = int(value)
		case 2:
			schema, err := readParquetSchemaList(reader, field.Type)
			if err != nil {
				return metadata, err
			}
			metadata.SchemaColumns = flattenParquetSchema(schema)
		case 3:
			value, err := reader.readI64()
			if err != nil {
				return metadata, err
			}
			metadata.NumRows = value
		case 4:
			rowGroups, err := readParquetRowGroupList(reader, field.Type)
			if err != nil {
				return metadata, err
			}
			metadata.RowGroups = rowGroups
		case 6:
			value, err := reader.readBinaryString()
			if err != nil {
				return metadata, err
			}
			metadata.CreatedBy = value
		default:
			if err := reader.skip(field.Type); err != nil {
				return metadata, err
			}
		}
	}
	for index := range metadata.RowGroups {
		metadata.RowGroups[index].Index = index + 1
	}
	return metadata, nil
}

func readParquetSchemaList(reader *compactReader, fieldType byte) ([]parquetSchemaElement, error) {
	if fieldType != compactList {
		return nil, fmt.Errorf("parquet schema field has compact type %d, want list", fieldType)
	}
	elementType, size, err := reader.readListHeader()
	if err != nil {
		return nil, err
	}
	if elementType != compactStruct {
		return nil, fmt.Errorf("parquet schema list element has compact type %d, want struct", elementType)
	}
	schema := make([]parquetSchemaElement, 0, size)
	for index := 0; index < size; index++ {
		element, err := readParquetSchemaElement(reader)
		if err != nil {
			return nil, err
		}
		schema = append(schema, element)
	}
	return schema, nil
}

func readParquetSchemaElement(reader *compactReader) (parquetSchemaElement, error) {
	element := parquetSchemaElement{}
	reader.pushStruct()
	defer reader.popStruct()
	for {
		field, err := reader.readFieldHeader()
		if err != nil {
			return element, err
		}
		if field.Type == compactStop {
			return element, nil
		}
		switch field.ID {
		case 1:
			value, err := reader.readI32()
			if err != nil {
				return element, err
			}
			element.Type = parquetPhysicalType(value)
		case 2:
			value, err := reader.readI32()
			if err != nil {
				return element, err
			}
			element.TypeLength = int(value)
		case 3:
			value, err := reader.readI32()
			if err != nil {
				return element, err
			}
			element.RepetitionType = parquetRepetitionType(value)
		case 4:
			value, err := reader.readBinaryString()
			if err != nil {
				return element, err
			}
			element.Name = value
		case 5:
			value, err := reader.readI32()
			if err != nil {
				return element, err
			}
			element.NumChildren = int(value)
		case 6:
			value, err := reader.readI32()
			if err != nil {
				return element, err
			}
			element.ConvertedType = parquetConvertedType(value)
		case 7:
			value, err := reader.readI32()
			if err != nil {
				return element, err
			}
			element.Scale = int(value)
		case 8:
			value, err := reader.readI32()
			if err != nil {
				return element, err
			}
			element.Precision = int(value)
		default:
			if err := reader.skip(field.Type); err != nil {
				return element, err
			}
		}
	}
}

func flattenParquetSchema(schema []parquetSchemaElement) []ParquetColumn {
	if len(schema) == 0 {
		return nil
	}
	columns := []ParquetColumn{}
	var walk func(index int, path []string) int
	walk = func(index int, path []string) int {
		if index >= len(schema) {
			return index
		}
		element := schema[index]
		nextPath := appendPath(path, element.Name)
		index++
		if element.NumChildren > 0 {
			for child := 0; child < element.NumChildren && index < len(schema); child++ {
				index = walk(index, nextPath)
			}
			return index
		}
		columns = append(columns, ParquetColumn{
			Path:           strings.Join(nextPath, "."),
			Type:           element.Type,
			RepetitionType: element.RepetitionType,
			ConvertedType:  element.ConvertedType,
			TypeLength:     element.TypeLength,
			Precision:      element.Precision,
			Scale:          element.Scale,
		})
		return index
	}
	start := 0
	if schema[0].NumChildren > 0 {
		start = 1
		for child := 0; child < schema[0].NumChildren && start < len(schema); child++ {
			start = walk(start, nil)
		}
		return columns
	}
	for start < len(schema) {
		start = walk(start, nil)
	}
	return columns
}

func appendPath(path []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return append([]string{}, path...)
	}
	next := append([]string{}, path...)
	return append(next, value)
}

func readParquetRowGroupList(reader *compactReader, fieldType byte) ([]ParquetRowGroup, error) {
	if fieldType != compactList {
		return nil, fmt.Errorf("parquet row_groups field has compact type %d, want list", fieldType)
	}
	elementType, size, err := reader.readListHeader()
	if err != nil {
		return nil, err
	}
	if elementType != compactStruct {
		return nil, fmt.Errorf("parquet row_groups list element has compact type %d, want struct", elementType)
	}
	rowGroups := make([]ParquetRowGroup, 0, size)
	for index := 0; index < size; index++ {
		rowGroup, err := readParquetRowGroup(reader)
		if err != nil {
			return nil, err
		}
		rowGroups = append(rowGroups, rowGroup)
	}
	return rowGroups, nil
}

func readParquetRowGroup(reader *compactReader) (ParquetRowGroup, error) {
	rowGroup := ParquetRowGroup{}
	reader.pushStruct()
	defer reader.popStruct()
	for {
		field, err := reader.readFieldHeader()
		if err != nil {
			return rowGroup, err
		}
		if field.Type == compactStop {
			return rowGroup, nil
		}
		switch field.ID {
		case 1:
			chunks, err := readParquetColumnChunkList(reader, field.Type)
			if err != nil {
				return rowGroup, err
			}
			rowGroup.ColumnChunks = chunks
			rowGroup.Columns = len(chunks)
			for _, chunk := range chunks {
				rowGroup.TotalCompressedSize += chunk.CompressedSize
				rowGroup.TotalUncompressedSize += chunk.UncompressedSize
			}
		case 2:
			value, err := reader.readI64()
			if err != nil {
				return rowGroup, err
			}
			rowGroup.TotalByteSize = value
		case 3:
			value, err := reader.readI64()
			if err != nil {
				return rowGroup, err
			}
			rowGroup.Rows = value
		case 6:
			value, err := reader.readI64()
			if err != nil {
				return rowGroup, err
			}
			rowGroup.TotalCompressedSize = value
		default:
			if err := reader.skip(field.Type); err != nil {
				return rowGroup, err
			}
		}
	}
}

func readParquetColumnChunkList(reader *compactReader, fieldType byte) ([]ParquetColumnChunk, error) {
	if fieldType != compactList {
		return nil, fmt.Errorf("parquet columns field has compact type %d, want list", fieldType)
	}
	elementType, size, err := reader.readListHeader()
	if err != nil {
		return nil, err
	}
	if elementType != compactStruct {
		return nil, fmt.Errorf("parquet columns list element has compact type %d, want struct", elementType)
	}
	chunks := make([]ParquetColumnChunk, 0, size)
	for index := 0; index < size; index++ {
		chunk, err := readParquetColumnChunk(reader)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(chunk.Path) != "" || chunk.Values > 0 || chunk.CompressedSize > 0 {
			chunks = append(chunks, chunk)
		}
	}
	return chunks, nil
}

func readParquetColumnChunk(reader *compactReader) (ParquetColumnChunk, error) {
	chunk := ParquetColumnChunk{}
	reader.pushStruct()
	defer reader.popStruct()
	for {
		field, err := reader.readFieldHeader()
		if err != nil {
			return chunk, err
		}
		if field.Type == compactStop {
			return chunk, nil
		}
		switch field.ID {
		case 3:
			metadata, err := readParquetColumnMetadata(reader, field.Type)
			if err != nil {
				return chunk, err
			}
			chunk = metadata
		default:
			if err := reader.skip(field.Type); err != nil {
				return chunk, err
			}
		}
	}
}

func readParquetColumnMetadata(reader *compactReader, fieldType byte) (ParquetColumnChunk, error) {
	if fieldType != compactStruct {
		return ParquetColumnChunk{}, fmt.Errorf("parquet column metadata has compact type %d, want struct", fieldType)
	}
	chunk := ParquetColumnChunk{}
	reader.pushStruct()
	defer reader.popStruct()
	for {
		field, err := reader.readFieldHeader()
		if err != nil {
			return chunk, err
		}
		if field.Type == compactStop {
			return chunk, nil
		}
		switch field.ID {
		case 1:
			value, err := reader.readI32()
			if err != nil {
				return chunk, err
			}
			chunk.Type = parquetPhysicalType(value)
		case 3:
			values, err := reader.readStringList(field.Type)
			if err != nil {
				return chunk, err
			}
			chunk.Path = strings.Join(values, ".")
		case 4:
			value, err := reader.readI32()
			if err != nil {
				return chunk, err
			}
			chunk.Codec = parquetCompressionCodec(value)
		case 5:
			value, err := reader.readI64()
			if err != nil {
				return chunk, err
			}
			chunk.Values = value
		case 6:
			value, err := reader.readI64()
			if err != nil {
				return chunk, err
			}
			chunk.UncompressedSize = value
		case 7:
			value, err := reader.readI64()
			if err != nil {
				return chunk, err
			}
			chunk.CompressedSize = value
		default:
			if err := reader.skip(field.Type); err != nil {
				return chunk, err
			}
		}
	}
}

package datasets

import "fmt"

func parquetPhysicalType(value int32) string {
	switch value {
	case 0:
		return "BOOLEAN"
	case 1:
		return "INT32"
	case 2:
		return "INT64"
	case 3:
		return "INT96"
	case 4:
		return "FLOAT"
	case 5:
		return "DOUBLE"
	case 6:
		return "BYTE_ARRAY"
	case 7:
		return "FIXED_LEN_BYTE_ARRAY"
	default:
		return fmt.Sprintf("TYPE_%d", value)
	}
}

func parquetRepetitionType(value int32) string {
	switch value {
	case 0:
		return "REQUIRED"
	case 1:
		return "OPTIONAL"
	case 2:
		return "REPEATED"
	default:
		return fmt.Sprintf("REPETITION_%d", value)
	}
}

func parquetConvertedType(value int32) string {
	switch value {
	case 0:
		return "UTF8"
	case 1:
		return "MAP"
	case 2:
		return "MAP_KEY_VALUE"
	case 3:
		return "LIST"
	case 4:
		return "ENUM"
	case 5:
		return "DECIMAL"
	case 6:
		return "DATE"
	case 7:
		return "TIME_MILLIS"
	case 8:
		return "TIME_MICROS"
	case 9:
		return "TIMESTAMP_MILLIS"
	case 10:
		return "TIMESTAMP_MICROS"
	case 11:
		return "UINT_8"
	case 12:
		return "UINT_16"
	case 13:
		return "UINT_32"
	case 14:
		return "UINT_64"
	case 15:
		return "INT_8"
	case 16:
		return "INT_16"
	case 17:
		return "INT_32"
	case 18:
		return "INT_64"
	case 19:
		return "JSON"
	case 20:
		return "BSON"
	case 21:
		return "INTERVAL"
	default:
		return fmt.Sprintf("CONVERTED_%d", value)
	}
}

func parquetCompressionCodec(value int32) string {
	switch value {
	case 0:
		return "UNCOMPRESSED"
	case 1:
		return "SNAPPY"
	case 2:
		return "GZIP"
	case 3:
		return "LZO"
	case 4:
		return "BROTLI"
	case 5:
		return "LZ4"
	case 6:
		return "ZSTD"
	case 7:
		return "LZ4_RAW"
	default:
		return fmt.Sprintf("CODEC_%d", value)
	}
}

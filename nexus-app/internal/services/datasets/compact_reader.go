package datasets

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

type compactField struct {
	ID   int16
	Type byte
}

type compactReader struct {
	data        []byte
	offset      int
	lastFieldID int16
	stack       []int16
}

func newCompactReader(data []byte) *compactReader {
	return &compactReader{data: data}
}

func (r *compactReader) pushStruct() {
	r.stack = append(r.stack, r.lastFieldID)
	r.lastFieldID = 0
}

func (r *compactReader) popStruct() {
	if len(r.stack) == 0 {
		r.lastFieldID = 0
		return
	}
	r.lastFieldID = r.stack[len(r.stack)-1]
	r.stack = r.stack[:len(r.stack)-1]
}

func (r *compactReader) readFieldHeader() (compactField, error) {
	header, err := r.readByte()
	if err != nil {
		return compactField{}, err
	}
	fieldType := header & 0x0f
	if fieldType == compactStop {
		return compactField{Type: compactStop}, nil
	}
	modifier := int16((header & 0xf0) >> 4)
	fieldID := int16(0)
	if modifier == 0 {
		value, err := r.readI16()
		if err != nil {
			return compactField{}, err
		}
		fieldID = value
	} else {
		fieldID = r.lastFieldID + modifier
	}
	r.lastFieldID = fieldID
	return compactField{ID: fieldID, Type: fieldType}, nil
}

func (r *compactReader) readByte() (byte, error) {
	if r.offset >= len(r.data) {
		return 0, errors.New("unexpected end of compact metadata")
	}
	value := r.data[r.offset]
	r.offset++
	return value, nil
}

func (r *compactReader) readI16() (int16, error) {
	value, err := r.readVarint()
	if err != nil {
		return 0, err
	}
	return int16(decodeZigZag(value)), nil
}

func (r *compactReader) readI32() (int32, error) {
	value, err := r.readVarint()
	if err != nil {
		return 0, err
	}
	return int32(decodeZigZag(value)), nil
}

func (r *compactReader) readI64() (int64, error) {
	value, err := r.readVarint()
	if err != nil {
		return 0, err
	}
	return decodeZigZag(value), nil
}

func (r *compactReader) readDouble() (float64, error) {
	bytes, err := r.readBytes(8)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(bytes)), nil
}

func (r *compactReader) readBinaryString() (string, error) {
	size, err := r.readVarint()
	if err != nil {
		return "", err
	}
	if size > uint64(len(r.data)-r.offset) {
		return "", errors.New("compact binary length exceeds remaining metadata")
	}
	bytes, err := r.readBytes(int(size))
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (r *compactReader) readStringList(fieldType byte) ([]string, error) {
	if fieldType != compactList {
		return nil, fmt.Errorf("compact value has type %d, want list", fieldType)
	}
	elementType, size, err := r.readListHeader()
	if err != nil {
		return nil, err
	}
	if elementType != compactBinary {
		return nil, fmt.Errorf("compact list element has type %d, want binary", elementType)
	}
	values := make([]string, 0, size)
	for index := 0; index < size; index++ {
		value, err := r.readBinaryString()
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (r *compactReader) readBytes(size int) ([]byte, error) {
	if size < 0 || r.offset+size > len(r.data) {
		return nil, errors.New("compact metadata read exceeds bounds")
	}
	bytes := r.data[r.offset : r.offset+size]
	r.offset += size
	return bytes, nil
}

func (r *compactReader) readVarint() (uint64, error) {
	var result uint64
	for shift := uint(0); shift < 64; shift += 7 {
		value, err := r.readByte()
		if err != nil {
			return 0, err
		}
		result |= uint64(value&0x7f) << shift
		if value&0x80 == 0 {
			return result, nil
		}
	}
	return 0, errors.New("compact varint is too long")
}

func (r *compactReader) readListHeader() (byte, int, error) {
	header, err := r.readByte()
	if err != nil {
		return 0, 0, err
	}
	size := int((header & 0xf0) >> 4)
	elementType := header & 0x0f
	if size == 15 {
		extended, err := r.readVarint()
		if err != nil {
			return 0, 0, err
		}
		if extended > uint64(len(r.data)) {
			return 0, 0, errors.New("compact list size exceeds metadata size")
		}
		size = int(extended)
	}
	return elementType, size, nil
}

func (r *compactReader) skip(fieldType byte) error {
	switch fieldType {
	case compactTrue, compactFalse:
		return nil
	case compactByte:
		_, err := r.readByte()
		return err
	case compactI16:
		_, err := r.readI16()
		return err
	case compactI32:
		_, err := r.readI32()
		return err
	case compactI64:
		_, err := r.readI64()
		return err
	case compactDouble:
		_, err := r.readDouble()
		return err
	case compactBinary:
		_, err := r.readBinaryString()
		return err
	case compactList, compactSet:
		elementType, size, err := r.readListHeader()
		if err != nil {
			return err
		}
		for index := 0; index < size; index++ {
			if err := r.skip(elementType); err != nil {
				return err
			}
		}
		return nil
	case compactMap:
		size, err := r.readVarint()
		if err != nil {
			return err
		}
		if size == 0 {
			return nil
		}
		typeByte, err := r.readByte()
		if err != nil {
			return err
		}
		keyType := (typeByte & 0xf0) >> 4
		valueType := typeByte & 0x0f
		for index := uint64(0); index < size; index++ {
			if err := r.skip(keyType); err != nil {
				return err
			}
			if err := r.skip(valueType); err != nil {
				return err
			}
		}
		return nil
	case compactStruct:
		r.pushStruct()
		defer r.popStruct()
		for {
			field, err := r.readFieldHeader()
			if err != nil {
				return err
			}
			if field.Type == compactStop {
				return nil
			}
			if err := r.skip(field.Type); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported compact type %d", fieldType)
	}
}

func decodeZigZag(value uint64) int64 {
	return int64(value>>1) ^ -int64(value&1)
}

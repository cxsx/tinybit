package binary

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
)

const (
	commandLength          = 12
	magicAndChecksumLength = 4
)

// Marshaler is the interface implemented by types that can marshal themselves into binary.
type Marshaler interface {
	MarshalBinary() ([]byte, error)
}

// Marshal returns the binary encoding of v.
func Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer

	if reflect.TypeOf(v).Kind() == reflect.Ptr {
		v = reflect.ValueOf(v).Elem().Interface()
	}

	switch v.(type) {
	case uint8, int32, uint32, int64, uint64, bool:
		if err := binary.Write(&buf, binary.LittleEndian, v); err != nil {
			return nil, err
		}

	// port
	case uint16:
		if err := binary.Write(&buf, binary.BigEndian, v); err != nil {
			return nil, err
		}

	case [magicAndChecksumLength]byte:
		magic, ok := v.([magicAndChecksumLength]byte)
		if !ok {
			return nil, fmt.Errorf("invalid magic or checksum: %v", v)
		}

		if _, err := buf.Write(magic[:]); err != nil {
			return nil, err
		}

	case [commandLength]byte:
		command, ok := v.([commandLength]byte)
		if !ok {
			return nil, fmt.Errorf("invalid command: %v", v)
		}

		if _, err := buf.Write(command[:]); err != nil {
			return nil, err
		}

	case []byte:
		bytes, ok := v.([]byte)
		if !ok {
			return nil, fmt.Errorf("invalid byte array: %v", v)
		}

		if _, err := buf.Write(bytes); err != nil {
			return nil, err
		}

	// VarStr.String
	case string:
		if _, err := buf.Write([]byte(v.(string))); err != nil {
			return nil, err
		}

	case Marshaler:
		b, err := v.(Marshaler).MarshalBinary()
		if err != nil {
			return nil, err
		}

		if _, err := buf.Write(b); err != nil {
			return nil, err
		}

	default:
		// is it a struct?
		if reflect.ValueOf(v).Kind() == reflect.Struct {
			b, err := marshalStruct(v)
			if err != nil {
				return nil, err
			}

			if _, err := buf.Write(b); err != nil {
				return nil, err
			}

			break
		}

		return nil, fmt.Errorf("unsupported type %s", reflect.TypeOf(v).String())
	}

	return buf.Bytes(), nil
}

func marshalStruct(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	vv := reflect.ValueOf(v)

	for i := 0; i < vv.NumField(); i++ {
		s, err := Marshal(reflect.Indirect(vv.Field(i)).Interface())
		if err != nil {
			f := reflect.TypeOf(v).Field(i).Name
			return nil, fmt.Errorf("failed to marshal field %s: %v", f, err)
		}

		if _, err := buf.Write(s); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
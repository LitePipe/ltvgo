package ltvgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// The LtvStruct is a type for holding generic LiteVector struct values.
// It is implemented as a lightweight wrapper around a `map[string]any`
// that records key insertion order. This is used for preserving the element
// order for generic LiteVectors, and can be used to traverse struct fields in wire order.
// Additionally, values may only be set once, preventing struct value duplication.
type LtvStruct struct {
	m    map[string]any
	keys []*string
}

func NewLtvStruct() *LtvStruct {
	return &LtvStruct{
		m: make(map[string]any),
	}
}

// Set a value and record its insertion value.
// If the key already exists in the map, an error is returned.
func (s *LtvStruct) Set(key string, value any) error {

	// // Keys must be non-zero length strings
	// if len(key) == 0 {
	// 	return errBadKey
	// }

	// // Keys cannot be duplicated
	// if _, exists := s.m[key]; exists {
	// 	return errDuplicateKey
	// }

	s.keys = append(s.keys, &key)
	s.m[key] = value
	return nil
}

// Get a value by key
func (s *LtvStruct) Get(key string) any {
	return s.m[key]
}

// Try to get a value by key.
// The returned boolean is true if the key was found,
// otherwise false is returned.
func (s *LtvStruct) TryGet(key string) (any, bool) {
	value, exists := s.m[key]
	return value, exists
}

// Print an LtbStruct in key order
func (s *LtvStruct) String() string {
	var sb strings.Builder
	sb.WriteString("{")

	for idx, key := range s.keys {
		if idx != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("\"")
		sb.WriteString(*key)
		sb.WriteString("\":")
		sb.WriteString(fmt.Sprint(s.m[*key]))
	}
	sb.WriteString("}")
	return sb.String()
}

// Serialize to a LiteVectors with ordered keys
func (s *LtvStruct) MarshalLTV() ([]byte, error) {
	var dataBuf bytes.Buffer
	e := NewStreamEncoder(&dataBuf)

	e.WriteStructStart()
	for _, key := range s.keys {
		e.WriteString(*key)

		valBuf, err := Marshal(s.m[*key])
		if err != nil {
			return nil, err
		}

		e.RawWrite(valBuf)
	}

	e.WriteStructEnd()

	return dataBuf.Bytes(), nil
}

// Deserialize from a LiteVector buffer
func (s *LtvStruct) UnmarshalLTV(buf []byte) error {

	d := NewDecoder(buf)

	value, err := d.Value()
	if err != nil {
		return err
	}

	newStruct, ok := value.(*LtvStruct)
	if !ok {
		return errExpectedStruct
	}

	*s = *newStruct
	return nil
}

// Serialize to JSON with ordered keys
func (s *LtvStruct) MarshalJSON() ([]byte, error) {

	var b bytes.Buffer
	b.WriteRune('{')

	for idx, key := range s.keys {
		if idx > 0 {
			b.WriteRune(',')
		}

		b.WriteRune('"')
		b.WriteString(*key)
		b.WriteString("\":")
		valBuf, err := json.Marshal(s.m[*key])
		if err != nil {
			return nil, err
		}
		b.Write(valBuf)
	}

	b.WriteRune('}')

	return b.Bytes(), nil
}

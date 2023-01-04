package ltvgo

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"unicode/utf8"
)

type LtvElementRole int

const defaultMaxValueLen = 1024 * 1024

const (
	RoleValue LtvElementRole = iota
	RoleListEnd
	RoleStructEnd
	RoleStructKey
	RoleStructValue
)

// A LiteVector element descriptor
type LtvElementDesc struct {
	Tag          byte
	TypeCode     TypeCode
	SizeCode     SizeCode
	Length       uint64
	TagOffset    int
	ValueOffset  int
	Role         LtvElementRole
	FirstElement bool
	Depth        int
}

type stackElement struct {
	code         TypeCode
	firstElement bool
}

type stackTracker struct {
	stack []stackElement
}

func (s *stackTracker) processTag(d *LtvElementDesc) error {

	d.FirstElement = true
	d.Depth = len(s.stack)
	d.Role = RoleValue

	if len(s.stack) > 0 {

		// Capture a flag for the first element in a list/struct
		d.FirstElement = s.stack[len(s.stack)-1].firstElement
		s.stack[len(s.stack)-1].firstElement = false

		if s.stack[len(s.stack)-1].code == Struct {
			// A struct start tag may only be followed by a string
			// or a struct end tag. If followed by a string, we toggle the
			// the stored tag in our stack to 'StructEnd' to indicate that we're
			// expecting a value to follow.
			if d.TypeCode == String {
				s.stack[len(s.stack)-1].code = End
				d.Role = RoleStructKey
			} else if d.TypeCode != End {
				return errBadKey
			}

		} else if s.stack[len(s.stack)-1].code == End {
			d.Role = RoleStructValue
			// In this state we're expecting a value type.
			if d.TypeCode == End {
				return errExpectedValue
			}
			s.stack[len(s.stack)-1].code = Struct
		}
	}

	// Push struct/list onto the nesting stack
	if d.TypeCode == Struct || d.TypeCode == List {

		// Max depth check
		if len(s.stack)+1 >= MaxNestingDepth {
			return errMaxNestingDepth
		}

		s.stack = append(s.stack, stackElement{code: d.TypeCode, firstElement: true})
	}

	// Pop struct/list from nesting stack
	if d.TypeCode == End {

		// Check for nesting type mismatch
		l := len(s.stack)
		if l == 0 {
			return errNestingMismatch
		}

		if s.stack[len(s.stack)-1].code == Struct {
			d.Role = RoleStructEnd
		} else if s.stack[len(s.stack)-1].code == List {
			d.Role = RoleListEnd
		}

		// Pop
		s.stack = s.stack[:len(s.stack)-1]
		d.Depth = len(s.stack)
	}

	return nil
}

type StreamDecoder struct {
	r io.Reader

	// Read offset in the byte stream
	offset int

	// Track struct/list nesting
	tracker stackTracker

	// If set to true, will return NOP tags.
	ReturnNops bool

	// The maximum length supported by the ReadValue function
	MaxValueLength uint64
}

func NewStreamDecoder(r io.Reader) *StreamDecoder {

	return &StreamDecoder{
		r:              r,
		offset:         0,
		ReturnNops:     false,
		MaxValueLength: defaultMaxValueLen,
	}
}

func NewStreamDecoderBytes(buf []byte) *StreamDecoder {
	return NewStreamDecoder(bytes.NewReader(buf))
}

// Read a byte from the underlying stream and return the byte or error.
func (s *StreamDecoder) ReadByte() (byte, error) {
	var buf [1]byte
	_, err := s.r.Read(buf[:])
	if err != nil {
		return 0, err
	}

	s.offset++
	return buf[0], err
}

// Read into buffer from the underlying stream.
func (s *StreamDecoder) Read(buf []byte) (int, error) {
	n, err := s.r.Read(buf)
	s.offset += n
	return n, err
}

// Read the full size of the buffer from the underlying stream.
func (s *StreamDecoder) ReadFull(buf []byte) error {
	n, err := io.ReadFull(s.r, buf)
	s.offset += n
	return err
}

// Read the next tag or tag and length prefix from the stream.
// On return, the scanner will be positioned over the value.
func (s *StreamDecoder) Next() (LtvElementDesc, error) {
	var buf [8]byte
	var d LtvElementDesc
	d.TagOffset = s.offset

	// Read the next byte/tag
	tag := NopTag
	for tag == NopTag {
		var err error
		tag, err = s.ReadByte()
		if err != nil {
			return d, err
		}

		d.Tag = tag
		if s.ReturnNops && tag == NopTag {
			return d, nil
		}
	}

	typeCode, SizeCode, err := decodeTag(tag)
	d.TypeCode = typeCode
	d.SizeCode = SizeCode
	if err != nil {
		return d, err
	}

	if err = s.tracker.processTag(&d); err != nil {
		return d, err
	}

	// Lookup the type size
	typeSize := uint64(d.TypeCode.Size())
	d.Length = typeSize
	d.ValueOffset = d.TagOffset

	if typeSize != 0 {
		d.ValueOffset += 1
	}

	// For a single element, we're done reading
	if d.SizeCode == SizeSingle {
		return d, nil
	}

	// For a vector element, read the length field
	lenSize := 1 << (d.SizeCode - Size1)

	err = s.ReadFull(buf[0:lenSize])
	if err != nil {
		return d, err
	}

	d.ValueOffset = d.TagOffset + 1 + lenSize

	// Decode the length out of the stream
	switch d.SizeCode {
	case Size1:
		d.Length = uint64(buf[0])
	case Size2:
		d.Length = uint64(binary.LittleEndian.Uint16(buf[0:lenSize]))
	case Size4:
		d.Length = uint64(binary.LittleEndian.Uint32(buf[0:lenSize]))
	case Size8:
		d.Length = binary.LittleEndian.Uint64(buf[0:lenSize])
	}

	// Validate length for type
	if d.Length%typeSize != 0 {
		return d, errInvalidVectorLen
	}

	// Return tag
	return d, nil
}

// Read the next value from a data stream.
func (s *StreamDecoder) Value() (any, error) {
	d, err := s.Next()
	if err != nil {
		return nil, err
	}
	return s.ReadValue(d)
}

// Read a single (generic) value from a data stream based on the passed descriptor.
func (s *StreamDecoder) ReadValue(d LtvElementDesc) (any, error) {

	// Handle single tag elements
	switch d.TypeCode {
	case Nil:
		return nil, nil
	case Struct:
		return s.ReadStruct()
	case List:
		return s.ReadList()
	case End:
		return nil, errExpectedValue
	}

	if d.Length > s.MaxValueLength {
		return nil, errMaxValueLenExceeded
	}

	// TODO: handle this buffer/allocating in a more performant manner
	val := make([]byte, d.Length)
	err := s.ReadFull(val)
	if err != nil {
		return nil, err
	}

	typeSize := d.TypeCode.Size()
	count := len(val) / typeSize

	switch d.TypeCode {
	case String:
		if !utf8.Valid(val) {
			return nil, errBadUtf8
		}
		return string(val), nil

	case Bool:
		if d.SizeCode == SizeSingle {
			return val[0] != 0, nil
		} else {
			vec := make([]bool, len(val))
			for i, v := range val {
				vec[i] = v != 0
			}
			return vec, nil
		}

	case U8:
		if d.SizeCode == SizeSingle {
			return uint8(val[0]), nil
		} else {
			return val, nil
		}

	case U16:
		if d.SizeCode == SizeSingle {
			return binary.LittleEndian.Uint16(val), nil
		} else {
			vec := make([]uint16, count)
			for i := 0; i < count; i++ {
				idx := i * typeSize
				vec[i] = binary.LittleEndian.Uint16(val[idx : idx+typeSize])
			}
			return vec, nil
		}

	case U32:
		if d.SizeCode == SizeSingle {
			return binary.LittleEndian.Uint32(val), nil
		} else {
			vec := make([]uint32, count)
			for i := 0; i < count; i++ {
				idx := i * typeSize
				vec[i] = binary.LittleEndian.Uint32(val[idx : idx+typeSize])
			}
			return vec, nil
		}

	case U64:
		if d.SizeCode == SizeSingle {
			return binary.LittleEndian.Uint64(val), nil
		} else {
			vec := make([]uint64, count)
			for i := 0; i < count; i++ {
				idx := i * typeSize
				vec[i] = binary.LittleEndian.Uint64(val[idx : idx+typeSize])
			}
			return vec, nil
		}

	case I8:
		if d.SizeCode == SizeSingle {
			return int8(val[0]), nil
		} else {
			vec := make([]int8, count)
			for i := 0; i < count; i++ {
				vec[i] = int8(val[i])
			}
			return vec, nil
		}

	case I16:
		if d.SizeCode == SizeSingle {
			return int16(binary.LittleEndian.Uint16(val)), nil
		} else {
			vec := make([]int16, count)
			for i := 0; i < count; i++ {
				idx := i * typeSize
				vec[i] = int16(binary.LittleEndian.Uint16(val[idx : idx+typeSize]))
			}
			return vec, nil
		}

	case I32:
		if d.SizeCode == SizeSingle {
			return int32(binary.LittleEndian.Uint32(val)), nil
		} else {
			vec := make([]int32, count)
			for i := 0; i < count; i++ {
				idx := i * typeSize
				vec[i] = int32(binary.LittleEndian.Uint32(val[idx : idx+typeSize]))
			}
			return vec, nil
		}

	case I64:
		if d.SizeCode == SizeSingle {
			return int64(binary.LittleEndian.Uint64(val)), nil
		} else {
			vec := make([]int64, count)
			for i := 0; i < count; i++ {
				idx := i * typeSize
				vec[i] = int64(binary.LittleEndian.Uint64(val[idx : idx+typeSize]))
			}
			return vec, nil
		}

	case F32:
		if d.SizeCode == SizeSingle {
			return math.Float32frombits(binary.LittleEndian.Uint32(val)), nil
		} else {
			vec := make([]float32, count)
			for i := 0; i < count; i++ {
				idx := i * typeSize
				vec[i] = math.Float32frombits(binary.LittleEndian.Uint32(val[idx : idx+typeSize]))
			}
			return vec, nil
		}

	case F64:
		if d.SizeCode == SizeSingle {
			return math.Float64frombits(binary.LittleEndian.Uint64(val)), nil
		} else {
			vec := make([]float64, count)
			for i := 0; i < count; i++ {
				idx := i * typeSize
				vec[i] = math.Float64frombits(binary.LittleEndian.Uint64(val[idx : idx+typeSize]))
			}
			return vec, nil
		}
	}

	// This should not be reachable
	panic("Expected value type")
}

// Read a list value from the data stream as a generic []any list.
func (s *StreamDecoder) ReadList() ([]any, error) {
	var l = make([]any, 0)

	for {
		desc, err := s.Next()
		if err != nil {
			return l, err
		}

		if desc.TypeCode == End {
			break
		}

		val, err := s.ReadValue(desc)
		if err != nil {
			return l, err
		}

		l = append(l, val)
	}

	return l, nil
}

// Read a struct value from the data stream as an LtvMap
func (s *StreamDecoder) ReadStruct() (*LtvStruct, error) {
	m := NewLtvStruct()

	for {
		desc, err := s.Next()
		if err != nil {
			return m, err
		}

		if desc.TypeCode == End {
			break
		}

		key, err := s.ReadValue(desc)
		if err != nil {
			return m, err
		}

		desc, err = s.Next()
		if err != nil {
			return m, err
		}

		value, err := s.ReadValue(desc)
		if err != nil {
			return m, err
		}

		err = m.Set(key.(string), value)
		if err != nil {
			return m, err
		}
	}

	return m, nil
}

// Skip the value of a tag
func (s *StreamDecoder) SkipValue(d LtvElementDesc) error {

	// TODO: reuse internal buffer
	var err error = nil
	var buf [64]byte

	n := int(d.Length)
	for n > 0 && err == nil {
		var nn, minN int
		minN = 64
		if n < minN {
			minN = n
		}

		nn, err = s.r.Read(buf[:minN])
		n -= nn
	}

	return err
}

// Skip the value currently under the scanner
func (s *StreamDecoder) Skip(d LtvElementDesc) error {

	// For a data element or vector with a fixed size, just move the stream position past it.
	if d.Length > 0 {
		s.SkipValue(d)
	}

	if d.TypeCode == List || d.TypeCode == Struct {
		startNest := len(s.tracker.stack)
		for startNest <= len(s.tracker.stack) {
			d, err := s.Next()
			if err != nil {
				return err
			}

			if d.Length > 0 {
				s.SkipValue(d)
			}
		}
	}

	return nil
}

// Skip the value currently under the scanner with additional validation checks.
// Current implementation is inefficient to the point of being an abomination.
// It should be refactored with fire...
func (s *StreamDecoder) ValidateAndSkip(d LtvElementDesc) error {

	// Check string validity
	if d.TypeCode == String {
		_, err := s.ReadValue(d)
		return err
	} else if d.TypeCode == List {
		for {
			desc, err := s.Next()
			if err != nil {
				return err
			}

			if desc.TypeCode == End {
				break
			}

			if err := s.ValidateAndSkip(desc); err != nil {
				return err
			}
		}
	} else if d.TypeCode == Struct {
		// Using a value map to keep track of, and validate keys.
		m := NewLtvStruct()
		for {
			desc, err := s.Next()
			if err != nil {
				return err
			}

			if desc.TypeCode == End {
				break
			}

			key, err := s.ReadValue(desc)
			if err != nil {
				return err
			}

			err = m.Set(key.(string), nil)
			if err != nil {
				return err
			}

			desc, err = s.Next()
			if err != nil {
				return err
			}

			if err := s.ValidateAndSkip(desc); err != nil {
				return err
			}
		}
	} else if d.Length > 0 {
		return s.SkipValue(d)
	}

	return nil
}

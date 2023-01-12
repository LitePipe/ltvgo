package ltvgo

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"math/bits"
	"unicode/utf8"
)

var (
	errBadSizeCode         = errors.New("ltv: size code out of range")
	errBadUtf8             = errors.New("ltv: string with invalid UTF-8 data")
	errBadKey              = errors.New("ltv: invalid struct key")
	errInvalidVectorLen    = errors.New("ltv: vector length invalid for data type")
	errMaxValueLenExceeded = errors.New("ltv: max value length exceeded")
	errMaxNestingDepth     = errors.New("ltv: max nesting depth exceeded")
	errNestingMismatch     = errors.New("ltv: mismatched struct/list end tags")
	errExpectedValue       = errors.New("ltv: expected value")
	errExpectedStruct      = errors.New("ltv: expected struct")
)

// Valid reports whether data is a valid LiteVector buffer.
func Valid(data []byte) bool {
	return Validate(data) == nil
}

func Validate(data []byte) error {
	d := NewDecoder(data)

	for {
		desc, err := d.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		err = d.ValidateAndSkip(desc)
		if err != nil {
			return err
		}
	}
}

// Maximum that structs/arrays can be nested in this library.
const MaxNestingDepth = 10000

// A LiteVector element descriptor
type LtvDesc struct {
	TypeCode TypeCode
	SizeCode SizeCode
	Length   uint64
	Offset   int
}

// Decode and validate a tag byte.
func decodeTag(tag byte) (TypeCode, SizeCode, error) {
	typeCode := TypeCode(tag >> 4)
	sizeCode := SizeCode(tag & 0x0F)

	// Check length code in valid range
	if sizeCode > Size8 || (typeCode <= End && sizeCode != SizeSingle) {
		return typeCode, sizeCode, errBadSizeCode
	}

	return typeCode, sizeCode, nil
}

// LiteVector element structure validation
// This includes:
// - String keys for struct field names
// - Missing struct values
// - Correct struct/list tag nesting
// - Maximum nesting depth
//
// s is a vector of TypeCodes that is used as a stack to store the
// element state. It can grow to MaxNestingDepth size.
func validateStructure(s []TypeCode, c TypeCode) ([]TypeCode, error) {

	// Struct form
	if len(s) > 0 {
		if s[len(s)-1] == Struct {
			// A struct start tag may only be followed by a string
			// or a struct end tag. If followed by a string, we toggle the
			// the stored tag in our stack to 'StructEnd' to indicate that we're
			// expecting a value to follow.
			if c == String {
				s[len(s)-1] = End
			} else if c != End {
				return s, errBadKey
			}
		} else if s[len(s)-1] == End {
			// In this state we're expecting a value type.
			if c == End {
				return s, errExpectedValue
			}
			s[len(s)-1] = Struct
		}
	}

	// Push struct/list onto the nesting stack
	if c == Struct || c == List {

		// Max depth check
		if len(s)+1 >= MaxNestingDepth {
			return s, errMaxNestingDepth
		}

		s = append(s, c)
	}

	// Pop struct/list from nesting stack
	if c == End {

		// Check for nesting type mismatch
		l := len(s)
		if l == 0 {
			return s, errNestingMismatch
		}

		// Pop
		s = s[:len(s)-1]
	}

	return s, nil
}

// A LiteVector buffer decoder
type Decoder struct {

	// The buffer we're scanning through
	buf []byte

	// Current position within the buffer
	pos int

	// Stack to keep track of struct/list nesting
	nStack []TypeCode
}

func NewDecoder(buf []byte) *Decoder {
	return &Decoder{
		buf:    buf,
		nStack: []TypeCode{},
		pos:    0,
	}
}

// Check whether x + y > bound with overflow checking.
func isInBound(x, y, bound uint64) bool {
	sum, carry := bits.Add64(x, y, 0)
	if carry == 1 {
		return false
	}

	return sum <= bound
}

// Read the next tag or tag and length prefix from the stream.
// On return, the scanner will be positioned over the value.
func (s *Decoder) Next() (LtvDesc, error) {

	var d LtvDesc

	// Scan past NOPs
	for s.pos < len(s.buf) && s.buf[s.pos] == NopTag {
		s.pos++
	}

	// We're at the end of buffer
	if s.pos == len(s.buf) {

		// Check for balanced nesting tags
		if len(s.nStack) != 0 {
			return d, io.ErrUnexpectedEOF
		}

		return d, io.EOF
	}

	// Mask out type and length codes from the tag byte
	typeCode, sizeCode, err := decodeTag(s.buf[s.pos])
	d.TypeCode = typeCode
	d.SizeCode = sizeCode
	d.Offset = s.pos

	if err != nil {
		return d, err
	}

	// Move position past the tag
	s.pos++

	// Lookup the type size
	typeSize := uint64(d.TypeCode.Size())
	d.Length = typeSize

	// Buffer size check
	if d.Length != 0 && !isInBound(uint64(s.pos), d.Length, uint64(len(s.buf))) {
		return d, io.ErrUnexpectedEOF
	}

	// Check element structure
	if s.nStack, err = validateStructure(s.nStack, d.TypeCode); err != nil {
		return d, err
	}

	// For a single element, we're done reading
	if d.SizeCode == SizeSingle {
		return d, nil
	}

	// For a vector element, read the length field
	lenSize := 1 << (d.SizeCode - Size1)

	// Buffer size check
	if !isInBound(uint64(s.pos), uint64(lenSize), uint64(len(s.buf))) {
		return d, io.ErrUnexpectedEOF
	}

	// Decode the length out of the stream
	switch d.SizeCode {
	case Size1:
		d.Length = uint64(s.buf[s.pos])
	case Size2:
		d.Length = uint64(binary.LittleEndian.Uint16(s.buf[s.pos:]))
	case Size4:
		d.Length = uint64(binary.LittleEndian.Uint32(s.buf[s.pos:]))
	case Size8:
		d.Length = binary.LittleEndian.Uint64(s.buf[s.pos:])
	}

	s.pos += lenSize

	// Validate length for type
	if d.Length%typeSize != 0 {
		return d, errInvalidVectorLen
	}

	// Buffer size check
	if !isInBound(uint64(s.pos), d.Length, uint64(len(s.buf))) {
		return d, io.ErrUnexpectedEOF
	}

	// Return tag ready to read a vector
	return d, nil
}

// Skip the value currently under the scanner with additional validation checks.
func (s *Decoder) ValidateAndSkip(d LtvDesc) error {

	// Check string validity
	if d.TypeCode == String {
		if !utf8.Valid(s.buf[s.pos : s.pos+int(d.Length)]) {
			return errBadUtf8
		}
	}

	// For a data element or vector with a fixed size, just move the stream position past it.
	if d.Length != 0 {
		s.pos += int(d.Length)
	}

	// Recursively process list items.
	if d.TypeCode == List {
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
	}

	// Recursively process struct items
	if d.TypeCode == Struct {
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
	}

	return nil
}

// Skip the value currently under the scanner
func (s *Decoder) Skip(d LtvDesc) error {

	// For a data element or vector with a fixed size, just move the stream position past it.
	s.pos += int(d.Length)

	if d.TypeCode == List || d.TypeCode == Struct {
		startNest := len(s.nStack)
		for startNest <= len(s.nStack) {
			d, err := s.Next()
			if err != nil {
				return err
			}
			s.pos += int(d.Length)
		}
	}

	return nil
}

// Read the next value from a data stream.
func (s *Decoder) Value() (any, error) {
	d, err := s.Next()
	if err != nil {
		return nil, err
	}
	return s.ReadValue(d)
}

// Read a single (generic) value from a data stream
// based on the given descriptor.
func (s *Decoder) ReadValue(d LtvDesc) (any, error) {

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

	// Slice the value length out of the buffer
	val := s.buf[s.pos : s.pos+int(d.Length)]
	s.pos += int(d.Length)

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
func (s *Decoder) ReadList() ([]any, error) {
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
func (s *Decoder) ReadStruct() (*LtvStruct, error) {
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

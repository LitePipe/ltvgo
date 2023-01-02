package ltvgo

import (
	"bufio"
	"encoding/binary"
	"io"
)

type LtvElementRole int

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
	r *bufio.Reader

	// Read offset in the byte stream
	offset int

	// Track struct/list nesting
	tracker stackTracker

	// If set to true, will return NOP tags.
	ReturnNops bool
}

func NewStreamDecoder(r io.Reader) *StreamDecoder {
	return &StreamDecoder{
		r:          bufio.NewReader(r),
		offset:     0,
		ReturnNops: false,
	}
}

// Read a byte from the underlying stream and return the byte or error.
func (s *StreamDecoder) ReadByte() (byte, error) {
	b, err := s.r.ReadByte()
	if err == nil {
		s.offset++
	}
	return b, err
}

// Read into buffer from the underlying stream.
func (s *StreamDecoder) Read(buf []byte) (int, error) {
	n, err := s.r.Read(buf)
	if err == nil {
		s.offset += n
	}
	return n, err
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

	n, err := s.Read(buf[0:lenSize])
	if err != nil || n != lenSize {
		return d, io.ErrUnexpectedEOF
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

// Skip the value of a tag
func (s *StreamDecoder) SkipValue(d LtvElementDesc) error {
	n, err := s.r.Discard(int(d.Length))
	if err != nil {
		return err
	}
	s.offset += n
	return nil
}

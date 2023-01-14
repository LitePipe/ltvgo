package ltvgo

import (
	"encoding/binary"
	"io"
	"math"
	"unicode/utf8"
)

type StreamEncoder struct {
	w       io.Writer
	offset  int
	Werr    error
	scratch [8]byte
}

func NewStreamEncoder(w io.Writer) *StreamEncoder {
	return &StreamEncoder{
		w:    w,
		Werr: nil,
	}
}

// Reset the error state and offset of the encoder.
func (e *StreamEncoder) Reset() {
	e.offset = 0
	e.Werr = nil
}

// Manually set the "offset" of the stream encoder.
func (e *StreamEncoder) SetOffset(offset int) {
	e.offset = offset
}

// Passthrough write a byte array to the underlying writer
// with error caching and offset tracking.
func (e *StreamEncoder) RawWrite(b []byte) {

	// Abort any writes if there is already an error in the stream.
	if e.Werr != nil {
		return
	}

	n, err := e.w.Write(b)
	e.Werr = err
	if err == nil {
		e.offset += n
	}
}

// Passthrough write byte
func (e *StreamEncoder) RawWriteByte(v byte) {
	e.scratch[0] = v
	e.RawWrite(e.scratch[:1])
}

// Passthrough write Uint16 endian corrected
func (e *StreamEncoder) RawWriteUint16(v uint16) {
	binary.LittleEndian.PutUint16(e.scratch[:], v)
	e.RawWrite(e.scratch[0:2])
}

// Passthrough write Uint32, endian corrected
func (e *StreamEncoder) RawWriteUint32(v uint32) {
	binary.LittleEndian.PutUint32(e.scratch[:], v)
	e.RawWrite(e.scratch[0:4])
}

// Passthrough write Uint64, endian corrected
func (e *StreamEncoder) RawWriteUint64(v uint64) {
	binary.LittleEndian.PutUint64(e.scratch[:], v)
	e.RawWrite(e.scratch[0:8])
}

func (e *StreamEncoder) WriteTag(t TypeCode, s SizeCode) {
	e.RawWriteByte((byte(t) << 4) | byte(s))
}

// Write the tag and length for a typed vector.
func (e *StreamEncoder) WriteVectorPrefix(t TypeCode, count int) {

	typeSize := typeSizes[t]
	bufLen := uint64(typeSize * count)

	// Compute the number of bytes 2^n needed to store a value and returns n.
	// Range 0..3
	var exp uint8
	switch {
	case bufLen <= math.MaxUint8:
		exp = 0
	case bufLen <= math.MaxUint16:
		exp = 1
	case bufLen <= math.MaxUint32:
		exp = 2
	default:
		exp = 3
	}

	lenCode := SizeCode(uint8(Size1) + exp)
	lenSize := 1 << exp

	// Alignment padding
	alignmentDelta := (e.offset + 1 + lenSize) & (typeSize - 1)
	if alignmentDelta != 0 {
		paddingLen := typeSize - alignmentDelta
		for i := 0; i < paddingLen; i++ {
			e.WriteNop()
		}
	}

	// Tag
	e.WriteTag(t, lenCode)

	// Length
	switch lenCode {
	case Size1:
		e.RawWriteByte(uint8(bufLen))
	case Size2:
		e.RawWriteUint16(uint16(bufLen))
	case Size4:
		e.RawWriteUint32(uint32(bufLen))
	case Size8:
		e.RawWriteUint64(uint64(bufLen))
	}
}

func (e *StreamEncoder) WriteNop() {
	e.RawWriteByte(NopTag)
}

func (e *StreamEncoder) WriteNil() {
	e.WriteTag(Nil, SizeSingle)
}

func (e *StreamEncoder) WriteStructStart() {
	e.WriteTag(Struct, SizeSingle)
}

func (e *StreamEncoder) WriteStructEnd() {
	e.WriteTag(End, SizeSingle)
}

func (e *StreamEncoder) WriteListStart() {
	e.WriteTag(List, SizeSingle)
}

func (e *StreamEncoder) WriteListEnd() {
	e.WriteTag(End, SizeSingle)
}

func (e *StreamEncoder) WriteBool(v bool) {
	e.WriteTag(Bool, SizeSingle)

	if v {
		e.RawWriteByte(1)
	} else {
		e.RawWriteByte(0)
	}
}

func (e *StreamEncoder) WriteI8(v int8) {
	e.WriteTag(I8, SizeSingle)
	e.RawWriteByte(uint8(v))
}

func (e *StreamEncoder) WriteI16(v int16) {
	e.WriteTag(I16, SizeSingle)
	e.RawWriteUint16(uint16(v))
}

func (e *StreamEncoder) WriteI32(v int32) {
	e.WriteTag(I32, SizeSingle)
	e.RawWriteUint32(uint32(v))
}

func (e *StreamEncoder) WriteI64(v int64) {
	e.WriteTag(I64, SizeSingle)
	e.RawWriteUint64(uint64(v))
}

func (e *StreamEncoder) WriteU8(v uint8) {
	e.WriteTag(U8, SizeSingle)
	e.RawWriteByte(v)
}

func (e *StreamEncoder) WriteU16(v uint16) {
	e.WriteTag(U16, SizeSingle)
	e.RawWriteUint16(v)
}

func (e *StreamEncoder) WriteU32(v uint32) {
	e.WriteTag(U32, SizeSingle)
	e.RawWriteUint32(v)
}

func (e *StreamEncoder) WriteU64(v uint64) {
	e.WriteTag(U64, SizeSingle)
	e.RawWriteUint64(v)
}

func (e *StreamEncoder) WriteF32(v float32) {
	e.WriteTag(F32, SizeSingle)
	e.RawWriteUint32(math.Float32bits(v))
}

func (e *StreamEncoder) WriteF64(v float64) {
	e.WriteTag(F64, SizeSingle)
	e.RawWriteUint64(math.Float64bits(v))
}

// Write int with Goldilocks fitting
func (e *StreamEncoder) WriteInt(v int64) {
	switch {
	case v >= math.MinInt8 && v <= math.MaxInt8:
		e.WriteI8(int8(v))
	case v >= math.MinInt16 && v <= math.MaxInt16:
		e.WriteI16(int16(v))
	case v >= math.MinInt32 && v <= math.MaxInt32:
		e.WriteI32(int32(v))
	default:
		e.WriteI64(v)
	}
}

// Write uint with Goldilocks fitting
func (e *StreamEncoder) WriteUint(v uint64) {
	switch {
	case v <= math.MaxUint8:
		e.WriteU8(uint8(v))
	case v <= math.MaxUint16:
		e.WriteU16(uint16(v))
	case v <= math.MaxUint32:
		e.WriteU32(uint32(v))
	default:
		e.WriteU64(v)
	}
}

////////////////////////////////////////////////////////////////////////////////

func (e *StreamEncoder) WriteString(s string) {

	// Validation up front
	if !utf8.ValidString(s) {
		panic("ltv: WriteString requires a valid UTF-8 string")
	}

	if len(s) == 1 {
		e.WriteTag(String, SizeSingle)
		e.RawWriteByte(byte(s[0]))
	} else {
		e.WriteVectorPrefix(String, len(s))
		e.RawWrite([]byte(s))
	}
}

func (e *StreamEncoder) WriteBoolVec(v []bool) {
	e.WriteVectorPrefix(Bool, len(v))
	for _, v := range v {
		if v {
			e.RawWriteByte(1)
		} else {
			e.RawWriteByte(0)
		}
	}
}

func (e *StreamEncoder) WriteBytes(v []byte) {
	e.WriteVectorPrefix(U8, len(v))
	e.RawWrite(v)
}

func (e *StreamEncoder) WriteU8Vec(v []uint8) {
	e.WriteVectorPrefix(U8, len(v))
	e.RawWrite(v)
}

func (e *StreamEncoder) WriteU16Vec(v []uint16) {
	e.WriteVectorPrefix(U16, len(v))
	for _, val := range v {
		e.RawWriteUint16(val)
	}
}

func (e *StreamEncoder) WriteU32Vec(v []uint32) {
	e.WriteVectorPrefix(U32, len(v))
	for _, val := range v {
		e.RawWriteUint32(val)
	}
}

func (e *StreamEncoder) WriteU64Vec(v []uint64) {
	e.WriteVectorPrefix(U64, len(v))
	for _, val := range v {
		e.RawWriteUint64(val)
	}
}

func (e *StreamEncoder) WriteI8Vec(v []int8) {
	e.WriteVectorPrefix(I8, len(v))
	for _, val := range v {
		e.RawWriteByte(byte(val))
	}
}

func (e *StreamEncoder) WriteI16Vec(v []int16) {
	e.WriteVectorPrefix(I16, len(v))
	for _, val := range v {
		e.RawWriteUint16(uint16(val))
	}
}

func (e *StreamEncoder) WriteI32Vec(v []int32) {
	e.WriteVectorPrefix(I32, len(v))
	for _, val := range v {
		e.RawWriteUint32(uint32(val))
	}
}

func (e *StreamEncoder) WriteI64Vec(v []int64) {
	e.WriteVectorPrefix(I64, len(v))
	for _, val := range v {
		e.RawWriteUint64(uint64(val))
	}
}

func (e *StreamEncoder) WriteF32Vec(v []float32) {
	e.WriteVectorPrefix(F32, len(v))
	for _, val := range v {
		e.RawWriteUint32(math.Float32bits(val))
	}
}

func (e *StreamEncoder) WriteF64Vec(v []float64) {
	e.WriteVectorPrefix(F64, len(v))
	for _, val := range v {
		e.RawWriteUint64(math.Float64bits(val))
	}
}

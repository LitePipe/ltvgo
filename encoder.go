package ltvgo

import (
	"encoding/binary"
	"math"
	"unicode/utf8"
)

const initialBufferSize = 64

type Encoder struct {
	buf     []byte // The buffer holding serialized data
	scratch [8]byte
}

func NewEncoder() *Encoder {
	return &Encoder{
		buf: make([]byte, 0, initialBufferSize),
	}
}

// Return the underlying buffer. This is only
// valid until the next buffer modification.
// Callers that want to hold onto this value should make a copy.
func (e *Encoder) Bytes() []byte {
	return e.buf
}

func (e *Encoder) Reset() {
	e.buf = e.buf[:0]
}

// Grow the buffer to accommodate new data.
// Returns the index where data should start being written.
func (e *Encoder) grow(n int) int {
	l := len(e.buf)

	if n <= cap(e.buf)-l {
		// Reslice to expand our length if we can
		e.buf = e.buf[:l+n]
	} else {
		// Otherwise, grow the buffer, letting append determine how much
		// capacity to (over)allocate.
		e.buf = append(e.buf, make([]byte, n)...)
	}

	return l
}

func (e *Encoder) RawWriteByte(data byte) {
	e.buf = append(e.buf, data)
}

func (e *Encoder) RawWrite(data []byte) {
	e.buf = append(e.buf, data...)
}

// Passthrough write Uint16 endian corrected
func (e *Encoder) RawWriteUint16(v uint16) {
	binary.LittleEndian.PutUint16(e.scratch[:], v)
	e.RawWrite(e.scratch[0:2])
}

// Passthrough write Uint32, endian corrected
func (e *Encoder) RawWriteUint32(v uint32) {
	binary.LittleEndian.PutUint32(e.scratch[:], v)
	e.RawWrite(e.scratch[0:4])
}

// Passthrough write Uint64, endian corrected
func (e *Encoder) RawWriteUint64(v uint64) {
	binary.LittleEndian.PutUint64(e.scratch[:], v)
	e.RawWrite(e.scratch[0:8])
}

func (e *Encoder) WriteTag(t TypeCode, s SizeCode) {
	e.RawWriteByte((byte(t) << 4) | byte(s))
}

// Write the tag and length for a typed vector.
func (e *Encoder) WriteVectorPrefix(t TypeCode, count int) {

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
	alignmentDelta := (len(e.buf) + 1 + lenSize) & (typeSize - 1)
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

func (e *Encoder) WriteNop() {
	e.RawWriteByte(NopTag)
}

func (e *Encoder) WriteNil() {
	e.WriteTag(Nil, SizeSingle)
}

func (e *Encoder) WriteStructStart() {
	e.WriteTag(Struct, SizeSingle)
}

func (e *Encoder) WriteStructEnd() {
	e.WriteTag(End, SizeSingle)
}

func (e *Encoder) WriteListStart() {
	e.WriteTag(List, SizeSingle)
}

func (e *Encoder) WriteListEnd() {
	e.WriteTag(End, SizeSingle)
}

func (e *Encoder) WriteBool(v bool) {
	e.WriteTag(Bool, SizeSingle)

	if v {
		e.RawWriteByte(1)
	} else {
		e.RawWriteByte(0)
	}
}

func (e *Encoder) WriteI8(v int8) {
	e.WriteTag(I8, SizeSingle)
	e.RawWriteByte(uint8(v))
}

func (e *Encoder) WriteI16(v int16) {
	e.WriteTag(I16, SizeSingle)
	e.RawWriteUint16(uint16(v))
}

func (e *Encoder) WriteI32(v int32) {
	e.WriteTag(I32, SizeSingle)
	e.RawWriteUint32(uint32(v))
}

func (e *Encoder) WriteI64(v int64) {
	e.WriteTag(I64, SizeSingle)
	e.RawWriteUint64(uint64(v))
}

func (e *Encoder) WriteU8(v uint8) {
	e.WriteTag(U8, SizeSingle)
	e.RawWriteByte(v)
}

func (e *Encoder) WriteU16(v uint16) {
	e.WriteTag(U16, SizeSingle)
	e.RawWriteUint16(v)
}

func (e *Encoder) WriteU32(v uint32) {
	e.WriteTag(U32, SizeSingle)
	e.RawWriteUint32(v)
}

func (e *Encoder) WriteU64(v uint64) {
	e.WriteTag(U64, SizeSingle)
	e.RawWriteUint64(v)
}

func (e *Encoder) WriteF32(v float32) {
	e.WriteTag(F32, SizeSingle)
	e.RawWriteUint32(math.Float32bits(v))
}

func (e *Encoder) WriteF64(v float64) {
	e.WriteTag(F64, SizeSingle)
	e.RawWriteUint64(math.Float64bits(v))
}

// Write int with Goldilocks fitting
func (e *Encoder) WriteInt(v int64) {
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
func (e *Encoder) WriteUint(v uint64) {
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

func (e *Encoder) WriteString(s string) {

	// Validation up front
	if !utf8.ValidString(s) {
		panic("ltv: WriteString requires a valid UTF-8 string")
	}

	if len(s) == 1 {
		e.WriteTag(String, SizeSingle)
		e.RawWriteByte(byte(s[0]))
	} else {
		e.WriteVectorPrefix(String, len(s))
		idx := e.grow(len(s))
		copy(e.buf[idx:], s)
	}
}

func (e *Encoder) WriteBoolVec(v []bool) {
	e.WriteVectorPrefix(Bool, len(v))
	for _, v := range v {
		if v {
			e.RawWriteByte(1)
		} else {
			e.RawWriteByte(0)
		}
	}
}

func (e *Encoder) WriteBytes(v []byte) {
	e.WriteVectorPrefix(U8, len(v))
	e.RawWrite(v)
}

func (e *Encoder) WriteU8Vec(v []uint8) {
	e.WriteVectorPrefix(U8, len(v))
	e.RawWrite(v)
}

func (e *Encoder) WriteU16Vec(v []uint16) {
	e.WriteVectorPrefix(U16, len(v))
	typeSize := typeSizes[U16]
	idx := e.grow(len(v) * typeSize)
	for _, val := range v {
		binary.LittleEndian.PutUint16(e.buf[idx:idx+typeSize], val)
		idx += typeSize
	}
}

func (e *Encoder) WriteU32Vec(v []uint32) {
	e.WriteVectorPrefix(U32, len(v))
	typeSize := typeSizes[U32]
	idx := e.grow(len(v) * typeSize)
	for _, val := range v {
		binary.LittleEndian.PutUint32(e.buf[idx:idx+typeSize], val)
		idx += typeSize
	}
}

func (e *Encoder) WriteU64Vec(v []uint64) {
	e.WriteVectorPrefix(U64, len(v))
	typeSize := typeSizes[U64]
	idx := e.grow(len(v) * typeSize)
	for _, val := range v {
		binary.LittleEndian.PutUint64(e.buf[idx:idx+typeSize], val)
		idx += typeSize
	}
}

func (e *Encoder) WriteI8Vec(v []int8) {
	e.WriteVectorPrefix(I8, len(v))
	typeSize := typeSizes[I8]
	idx := e.grow(len(v) * typeSize)
	for _, val := range v {
		e.buf[idx] = byte(val)
		idx += typeSize
	}
}

func (e *Encoder) WriteI16Vec(v []int16) {
	e.WriteVectorPrefix(I16, len(v))
	typeSize := typeSizes[I16]
	idx := e.grow(len(v) * typeSize)
	for _, val := range v {
		binary.LittleEndian.PutUint16(e.buf[idx:idx+typeSize], uint16(val))
		idx += typeSize
	}
}

func (e *Encoder) WriteI32Vec(v []int32) {
	e.WriteVectorPrefix(I32, len(v))
	typeSize := typeSizes[I32]
	idx := e.grow(len(v) * typeSize)
	for _, val := range v {
		binary.LittleEndian.PutUint32(e.buf[idx:idx+typeSize], uint32(val))
		idx += typeSize
	}
}

func (e *Encoder) WriteI64Vec(v []int64) {
	e.WriteVectorPrefix(I64, len(v))
	typeSize := typeSizes[I64]
	idx := e.grow(len(v) * typeSize)
	for _, val := range v {
		binary.LittleEndian.PutUint64(e.buf[idx:idx+typeSize], uint64(val))
		idx += typeSize
	}
}

func (e *Encoder) WriteF32Vec(v []float32) {
	e.WriteVectorPrefix(F32, len(v))
	typeSize := typeSizes[F32]
	idx := e.grow(len(v) * typeSize)
	for _, val := range v {
		binary.LittleEndian.PutUint32(e.buf[idx:idx+typeSize], math.Float32bits(val))
		idx += typeSize
	}
}

func (e *Encoder) WriteF64Vec(v []float64) {
	e.WriteVectorPrefix(F64, len(v))
	typeSize := typeSizes[F64]
	idx := e.grow(len(v) * typeSize)
	for _, val := range v {
		binary.LittleEndian.PutUint64(e.buf[idx:idx+typeSize], math.Float64bits(val))
		idx += typeSize
	}
}

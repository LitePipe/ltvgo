package ltvgo

import (
	"bytes"
	"encoding/binary"
	"math"
	"unicode/utf8"
)

type Encoder struct {
	b       bytes.Buffer
	scratch [8]byte
}

func NewBufEncoder() *Encoder {
	return &Encoder{}
}

func (e *Encoder) Bytes() []byte {
	return e.b.Bytes()
}

func (e *Encoder) Reset() {
	e.b.Reset()
}

func (e *Encoder) Write(data []byte) {
	e.b.Write(data)
}

// Passthrough write Uint16 endian corrected
func (e *Encoder) RawWriteUint16(v uint16) {
	binary.LittleEndian.PutUint16(e.scratch[:], v)
	e.b.Write(e.scratch[0:2])
}

// Passthrough write Uint32, endian corrected
func (e *Encoder) RawWriteUint32(v uint32) {
	binary.LittleEndian.PutUint32(e.scratch[:], v)
	e.b.Write(e.scratch[0:4])
}

// Passthrough write Uint64, endian corrected
func (e *Encoder) RawWriteUint64(v uint64) {
	binary.LittleEndian.PutUint64(e.scratch[:], v)
	e.b.Write(e.scratch[0:8])
}

func (e *Encoder) WriteTag(t TypeCode, s SizeCode) {
	e.b.WriteByte((byte(t) << 4) | byte(s))
}

// Write the tag and length for a typed vector.
func (e *Encoder) WriteVectorPrefix(t TypeCode, count int) {

	typeSize := typeSizes[t]
	bufLen := uint64(typeSize * count)
	exp := fitStorageExponent(bufLen)
	lenCode := SizeCode(uint8(Size1) + exp)
	lenSize := 1 << exp
	alignmentDelta := (e.b.Len() + 1 + lenSize) & (typeSize - 1)

	// Alignment padding
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
		e.b.WriteByte(uint8(bufLen))
	case Size2:
		e.RawWriteUint16(uint16(bufLen))
	case Size4:
		e.RawWriteUint32(uint32(bufLen))
	case Size8:
		e.RawWriteUint64(uint64(bufLen))
	}
}

func (e *Encoder) WriteNop() {
	e.b.WriteByte(NopTag)
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
		e.b.WriteByte(1)
	} else {
		e.b.WriteByte(0)
	}
}

func (e *Encoder) WriteI8(v int8) {
	e.WriteTag(I8, SizeSingle)
	e.b.WriteByte(uint8(v))
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
	e.b.WriteByte(v)
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

func (e *Encoder) WriteInt(v int64) {
	// Goldilocks fit
	if v >= 0 {
		// Positive
		e.WriteUint(uint64(v))
	} else {
		// Negative
		u64 := uint64(v)
		switch {
		case u64&i8Mask == i8Mask:
			e.WriteI8(int8(v))
		case u64&i16Mask == i16Mask:
			e.WriteI16(int16(v))
		case u64&i32Mask == i32Mask:
			e.WriteI32(int32(v))
		default:
			e.WriteI64(v)
		}
	}
}

func (e *Encoder) WriteUint(v uint64) {
	// Goldilocks fit
	switch {
	case v&u8Mask == 0:
		e.WriteU8(uint8(v))
	case v&u16Mask == 0:
		e.WriteU16(uint16(v))
	case v&u32Mask == 0:
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
		e.b.WriteByte(byte(s[0]))
	} else {
		e.WriteVectorPrefix(String, len(s))
		e.b.WriteString(s)
	}
}

func (e *Encoder) WriteVecBool(v []bool) {
	e.WriteVectorPrefix(Bool, len(v))
	for _, v := range v {
		if v {
			e.b.WriteByte(1)
		} else {
			e.b.WriteByte(0)
		}
	}
}

func (e *Encoder) WriteBytes(v []byte) {
	e.WriteVectorPrefix(U8, len(v))
	e.b.Write(v)
}

func (e *Encoder) WriteVecU8(v []uint8) {
	e.WriteVectorPrefix(U8, len(v))
	e.b.Write(v)
}

func (e *Encoder) WriteVecU16(v []uint16) {
	e.WriteVectorPrefix(U16, len(v))
	for _, val := range v {
		e.RawWriteUint16(val)
	}
}

func (e *Encoder) WriteVecU32(v []uint32) {
	e.WriteVectorPrefix(U32, len(v))
	for _, val := range v {
		e.RawWriteUint32(val)
	}
}

func (e *Encoder) WriteVecU64(v []uint64) {
	e.WriteVectorPrefix(U64, len(v))
	for _, val := range v {
		e.RawWriteUint64(val)
	}
}

func (e *Encoder) WriteVecI8(v []int8) {
	e.WriteVectorPrefix(I8, len(v))
	for _, val := range v {
		e.b.WriteByte(byte(val))
	}
}

func (e *Encoder) WriteVecI16(v []int16) {
	e.WriteVectorPrefix(I16, len(v))
	for _, val := range v {
		e.RawWriteUint16(uint16(val))
	}
}

func (e *Encoder) WriteVecI32(v []int32) {
	e.WriteVectorPrefix(I32, len(v))
	for _, val := range v {
		e.RawWriteUint32(uint32(val))
	}
}

func (e *Encoder) WriteVecI64(v []int64) {
	e.WriteVectorPrefix(I64, len(v))
	for _, val := range v {
		e.RawWriteUint64(uint64(val))
	}
}

func (e *Encoder) WriteVecF32(v []float32) {
	e.WriteVectorPrefix(F32, len(v))
	for _, val := range v {
		e.RawWriteUint32(math.Float32bits(val))
	}
}

func (e *Encoder) WriteVecF64(v []float64) {
	e.WriteVectorPrefix(F64, len(v))
	for _, val := range v {
		e.RawWriteUint64(math.Float64bits(val))
	}
}

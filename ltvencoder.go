package ltvgo

// A interface for a generic LiteVector encoder
type LtvEncoder interface {
	WriteNop()
	WriteNil()
	WriteStructStart()
	WriteStructEnd()
	WriteListStart()
	WriteListEnd()
	WriteBool(bool)
	WriteU8(uint8)
	WriteU16(uint16)
	WriteU32(uint32)
	WriteU64(uint64)
	WriteI8(int8)
	WriteI16(int16)
	WriteI32(int32)
	WriteI64(int64)

	WriteF32(float32)
	WriteF64(float64)

	WriteInt(int64)
	WriteUint(uint64)
	WriteString(string)
	WriteBytes([]byte)

	WriteBoolVec([]bool)
	WriteU8Vec([]uint8)
	WriteU16Vec([]uint16)
	WriteU32Vec([]uint32)
	WriteU64Vec([]uint64)
	WriteI8Vec([]int8)
	WriteI16Vec([]int16)
	WriteI32Vec([]int32)
	WriteI64Vec([]int64)
	WriteF32Vec([]float32)
	WriteF64Vec([]float64)
}

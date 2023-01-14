package ltvgo

type StructEncoder struct {
	enc LtvEncoder
}

func NewStructEncoder(enc LtvEncoder) StructEncoder {
	return StructEncoder{
		enc: enc,
	}
}

func (s StructEncoder) StartStruct() {
	s.enc.WriteStructStart()
}

func (s StructEncoder) EndStruct() {
	s.enc.WriteStructEnd()
}

func (s StructEncoder) Nil(key string) {
	s.enc.WriteString(key)
	s.enc.WriteNil()
}

func (s StructEncoder) Bool(key string, v bool) {
	s.enc.WriteString(key)
	s.enc.WriteBool(v)
}

func (s StructEncoder) U8(key string, v uint8) {
	s.enc.WriteString(key)
	s.enc.WriteU8(v)
}

func (s StructEncoder) U16(key string, v uint16) {
	s.enc.WriteString(key)
	s.enc.WriteU16(v)
}

func (s StructEncoder) U32(key string, v uint32) {
	s.enc.WriteString(key)
	s.enc.WriteU32(v)
}

func (s StructEncoder) U64(key string, v uint64) {
	s.enc.WriteString(key)
	s.enc.WriteU64(v)
}

func (s StructEncoder) I8(key string, v int8) {
	s.enc.WriteString(key)
	s.enc.WriteI8(v)
}

func (s StructEncoder) I16(key string, v int16) {
	s.enc.WriteString(key)
	s.enc.WriteI16(v)
}

func (s StructEncoder) I32(key string, v int32) {
	s.enc.WriteString(key)
	s.enc.WriteI32(v)
}

func (s StructEncoder) I64(key string, v int64) {
	s.enc.WriteString(key)
	s.enc.WriteI64(v)
}

func (s StructEncoder) Int(key string, v int64) {
	s.enc.WriteString(key)
	s.enc.WriteInt(v)
}

func (s StructEncoder) Uint(key string, v uint64) {
	s.enc.WriteString(key)
	s.enc.WriteUint(v)
}

func (s StructEncoder) String(key string, v string) {
	s.enc.WriteString(key)
	s.enc.WriteString(v)
}

func (s StructEncoder) Bytes(key string, v []byte) {
	s.enc.WriteString(key)
	s.enc.WriteBytes(v)
}

func (s StructEncoder) BoolVec(key string, v []bool) {
	s.enc.WriteString(key)
	s.enc.WriteBoolVec(v)
}

func (s StructEncoder) U8Vec(key string, v []uint8) {
	s.enc.WriteString(key)
	s.enc.WriteU8Vec(v)
}

func (s StructEncoder) U16Vec(key string, v []uint16) {
	s.enc.WriteString(key)
	s.enc.WriteU16Vec(v)
}

func (s StructEncoder) U32Vec(key string, v []uint32) {
	s.enc.WriteString(key)
	s.enc.WriteU32Vec(v)
}

func (s StructEncoder) U64Vec(key string, v []uint64) {
	s.enc.WriteString(key)
	s.enc.WriteU64Vec(v)
}

func (s StructEncoder) I8Vec(key string, v []int8) {
	s.enc.WriteString(key)
	s.enc.WriteI8Vec(v)
}

func (s StructEncoder) I16Vec(key string, v []int16) {
	s.enc.WriteString(key)
	s.enc.WriteI16Vec(v)
}

func (s StructEncoder) I32Vec(key string, v []int32) {
	s.enc.WriteString(key)
	s.enc.WriteI32Vec(v)
}

func (s StructEncoder) I64Vec(key string, v []int64) {
	s.enc.WriteString(key)
	s.enc.WriteI64Vec(v)
}

func (s StructEncoder) F32Vec(key string, v []float32) {
	s.enc.WriteString(key)
	s.enc.WriteF32Vec(v)
}

func (s StructEncoder) F64Vec(key string, v []float64) {
	s.enc.WriteString(key)
	s.enc.WriteF64Vec(v)
}

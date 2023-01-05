package bench

import (
	"bytes"
	"testing"

	ltv "github.com/ThadThompson/ltvgo"
)

type LtvEncoder interface {
	WriteNop()
	WriteNil()
	WriteStructStart()
	WriteStructEnd()
	WriteListStart()
	WriteListEnd()
	WriteBool(bool)
	WriteI8(int8)
	WriteI16(int16)
	WriteI32(int32)
	WriteI64(int64)
	WriteU8(uint8)
	WriteU16(uint16)
	WriteU32(uint32)
	WriteU64(uint64)
	WriteF32(float32)
	WriteF64(float64)

	WriteInt(int64)
	WriteUint(uint64)
	WriteString(string)
	WriteVecBool([]bool)
	WriteVecU8([]uint8)
	WriteVecU16([]uint16)
	WriteVecU32([]uint32)
	WriteVecU64([]uint64)
	WriteVecI8([]int8)
	WriteVecI16([]int16)
	WriteVecI32([]int32)
	WriteVecI64([]int64)
	WriteVecF32([]float32)
	WriteVecF64([]float64)
}

func encodeSmall(e LtvEncoder) {
	e.WriteNop()
	e.WriteStructStart()
	e.WriteString("A")
	e.WriteI32(123)
	e.WriteStructEnd()
}

func encodeMedium(e LtvEncoder) {
	e.WriteStructStart()

	e.WriteString("An I32")
	e.WriteI32(123)

	e.WriteString("A list of things")
	e.WriteListStart()
	e.WriteF32(123.456)
	e.WriteF64(123456.7890)
	e.WriteI8(44)
	e.WriteBool(true)
	e.WriteString("fin")
	e.WriteListEnd()

	e.WriteString("A string")
	e.WriteString("Theory")

	e.WriteStructEnd()
}

func Benchmark_Encoder_Small(b *testing.B) {
	var e ltv.Encoder
	//e := ltv.NewBufEncoder()

	for i := 0; i < b.N; i++ {
		e.Reset()
		encodeSmall(&e)
	}
}

func Benchmark_EncoderStream_Small(b *testing.B) {
	var buf bytes.Buffer
	e := ltv.NewStreamEncoder(&buf)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		e.Reset()
		encodeSmall(e)
	}
}

func Benchmark_Encoder_Medium(b *testing.B) {
	var e ltv.Encoder
	//e := ltv.NewBufEncoder()

	for i := 0; i < b.N; i++ {
		e.Reset()
		encodeMedium(&e)
	}
}

func Benchmark_EncoderStream_Medium(b *testing.B) {
	var buf bytes.Buffer
	e := ltv.NewStreamEncoder(&buf)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		e.Reset()
		encodeMedium(e)
	}
}

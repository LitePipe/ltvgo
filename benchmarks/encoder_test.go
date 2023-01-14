package bench

import (
	"bytes"
	"testing"

	"github.com/ThadThompson/ltvgo"
	ltv "github.com/ThadThompson/ltvgo"
)

func encodeSmall(e ltvgo.LtvEncoder) {
	e.WriteNop()
	e.WriteStructStart()
	e.WriteString("A")
	e.WriteI32(123)
	e.WriteStructEnd()
}

func encodeMedium(e ltvgo.LtvEncoder) {
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

var floatVector = []float32{111.111, 222.222, 333.333, 444.444, 555.555}

func encodeVector(e ltvgo.LtvEncoder) {
	e.WriteF32Vec(floatVector)
}

func BenchmarkEncoderSmall(b *testing.B) {
	e := ltv.NewEncoder()

	for i := 0; i < b.N; i++ {
		e.Reset()
		encodeSmall(e)
	}
}

func BenchmarkStreamEncoderSmall(b *testing.B) {
	var buf bytes.Buffer
	e := ltv.NewStreamEncoder(&buf)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		e.Reset()
		encodeSmall(e)
	}
}

func BenchmarkEncoderMedium(b *testing.B) {
	e := ltv.NewEncoder()

	for i := 0; i < b.N; i++ {
		e.Reset()
		encodeMedium(e)
	}
}

func BenchmarkStreamEncoderMedium(b *testing.B) {
	var buf bytes.Buffer
	e := ltv.NewStreamEncoder(&buf)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		e.Reset()
		encodeMedium(e)
	}
}

func BenchmarkEncoderVector(b *testing.B) {
	e := ltv.NewEncoder()

	for i := 0; i < b.N; i++ {
		e.Reset()
		encodeVector(e)
	}
}

func BenchmarkStreamEncoderVector(b *testing.B) {
	var buf bytes.Buffer
	e := ltv.NewStreamEncoder(&buf)

	for i := 0; i < b.N; i++ {
		buf.Reset()
		e.Reset()
		encodeVector(e)
	}
}

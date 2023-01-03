package testvectors

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"math"

	ltv "github.com/ThadThompson/ltvgo"
	ltvjs "github.com/ThadThompson/ltvgo/json"
)

func GeneratePositiveVectors(w io.Writer) {

	// Setup
	var descBuf bytes.Buffer
	var dataBuf bytes.Buffer
	e := ltv.NewEncoder(&dataBuf)

	desc := func(s string) {
		descBuf.WriteString(s)
	}

	commit := func() {
		fmt.Fprintln(w, descBuf.String())
		fmt.Fprintln(w, hex.EncodeToString(dataBuf.Bytes()))

		// Reset state
		descBuf.Reset()
		dataBuf.Reset()
		e.Reset()
	}

	marshal := func(val any) {
		enc, err := ltv.Marshal(&val)
		if err != nil {
			panic("unable to Marshal round trip vector")
		}

		ltvjs.Ltv2Json(bytes.NewBuffer(enc), w, false)
		fmt.Fprintln(w)

		fmt.Fprintln(w, hex.EncodeToString(enc))
	}

	////////////////////////////////////////////////////////////////////////////////
	// Basic Types
	////////////////////////////////////////////////////////////////////////////////

	desc("nil")
	e.WriteNil()
	commit()

	desc("string: A")
	e.WriteString("A")
	commit()

	desc("bool: true")
	e.WriteBool(true)
	commit()

	desc("bool: false")
	e.WriteBool(false)
	commit()

	desc("u8:  123")
	e.WriteU8(123)
	commit()

	desc("u16: 1234")
	e.WriteU16(1234)
	commit()

	desc("u32: 123456")
	e.WriteU32(123456)
	commit()

	desc("u64: 123456789012")
	e.WriteU64(123456789012)
	commit()

	desc("i8:  -123")
	e.WriteI8(-123)
	commit()

	desc("i16: -1234")
	e.WriteI16(-1234)
	commit()

	desc("i32: -123456")
	e.WriteI32(-123456)
	commit()

	desc("i64: -123456789012")
	e.WriteI64(-123456789012)
	commit()

	desc("f32: 123.456")
	e.WriteF32(123.456)
	commit()

	desc("f64: 123456.789012")
	e.WriteF64(123456.789012)
	commit()

	////////////////////////////////////////////////////////////////////////////////
	// Vectors
	////////////////////////////////////////////////////////////////////////////////

	desc("string:  Hello World")
	e.WriteString("Hello World")
	commit()

	desc("bool[]: [true, false, false, true]")
	e.WriteVecBool([]bool{true, false, false, true})
	commit()

	desc("u8[]: [1, 2, 3, 4]")
	e.WriteVecU8([]uint8{1, 2, 3, 4})
	commit()

	desc("u16[]: [10, 20, 30, 40]")
	e.WriteVecU16([]uint16{10, 20, 30, 40})
	commit()

	desc("u32[]: [100, 200, 300, 400]")
	e.WriteVecU32([]uint32{100, 200, 300, 400})
	commit()

	desc("u64[]: [1000, 2000, 3000, 4000]")
	e.WriteVecU64([]uint64{1000, 2000, 3000, 4000})
	commit()

	desc("i8[]: [-1, -2, -3, -4]")
	e.WriteVecI8([]int8{-1, -2, -3, -4})
	commit()

	desc("i16[]: [-10, -20, -30, -0]")
	e.WriteVecI16([]int16{-10, -20, -30, -40})
	commit()

	desc("i32[]: [-100, -200, -300, -400]")
	e.WriteVecI32([]int32{-100, -200, -300, -400})
	commit()

	desc("i64[]: [-1000, -2000, -3000, -4000]")
	e.WriteVecI64([]int64{-1000, -2000, -3000, -4000})
	commit()

	desc("f32[]: [111.111, 222.222, 333.333, 444.444, 555.555]")
	e.WriteVecF32([]float32{111.111, 222.222, 333.333, 444.444, 555.555})
	commit()

	desc("f64[]: [111.111, 222.222, 333.333, 444.444, 555.555]")
	e.WriteVecF64([]float64{111.111, 222.222, 333.333, 444.444, 555.555})
	commit()

	////////////////////////////////////////////////////////////////////////////////
	// Containers
	////////////////////////////////////////////////////////////////////////////////

	desc("list: ['A', 123, false]")
	e.WriteListStart()
	e.WriteString("A")
	e.WriteInt(123)
	e.WriteBool(false)
	e.WriteListEnd()
	commit()

	desc("struct: {'A': 123, 'B': true}")
	e.WriteStructStart()
	e.WriteString("A")
	e.WriteInt(123)
	e.WriteString("B")
	e.WriteBool(true)
	e.WriteStructEnd()
	commit()

	////////////////////////////////////////////////////////////////////////////////
	// Boundaries
	////////////////////////////////////////////////////////////////////////////////

	// Boolean

	desc("bool: 2")
	e.WriteTag(ltv.Bool, ltv.SizeSingle)
	e.RawWriteByte(2)
	commit()

	desc("bool: 255")
	e.WriteTag(ltv.Bool, ltv.SizeSingle)
	e.RawWriteByte(255)
	commit()

	// Unsigned Integers

	desc("u8: 0")
	e.WriteU8(0)
	commit()

	desc("u8: 255")
	e.WriteU8(255)
	commit()

	desc("u16: 0")
	e.WriteU16(0)
	commit()

	desc("u16: 65535")
	e.WriteU16(65535)
	commit()

	desc("u32: 0")
	e.WriteU32(0)
	commit()

	desc("u32: 4294967295")
	e.WriteU32(4294967295)
	commit()

	desc("u64: 0")
	e.WriteU64(0)
	commit()

	desc("u64: 18446744073709551615")
	e.WriteU64(18446744073709551615)
	commit()

	// Signed Integers
	desc("i8: -128")
	e.WriteI8(-128)
	commit()

	desc("i8: 0")
	e.WriteI8(0)
	commit()

	desc("i8: 127")
	e.WriteI8(127)
	commit()

	desc("i16: -32768")
	e.WriteI16(-32768)
	commit()

	desc("i16_zero: 0")
	e.WriteI16(0)
	commit()

	desc("i16: 32767")
	e.WriteI16(32767)
	commit()

	desc("i32: -2147483648")
	e.WriteI32(-2147483648)
	commit()

	desc("i32: 0")
	e.WriteI32(0)
	commit()

	desc("i32: 2147483647")
	e.WriteI32(2147483647)
	commit()

	desc("i64: -9223372036854775808")
	e.WriteI64(-9223372036854775808)
	commit()

	desc("i64: 0")
	e.WriteI64(0)
	commit()

	desc("i64: 9223372036854775807")
	e.WriteI64(9223372036854775807)
	commit()

	// float 32-bit
	desc("f32: 0")
	e.WriteF32(0)
	commit()

	desc("f32: smallest non-zero")
	e.WriteF32(math.SmallestNonzeroFloat32)
	commit()

	desc("f32: MAX")
	e.WriteF32(math.MaxFloat32)
	commit()

	desc("f32: MIN")
	e.WriteF32(-math.MaxFloat32)
	commit()

	desc("f32: NaN")
	e.WriteF32(float32(math.NaN()))
	commit()

	desc("f32: +Inf")
	e.WriteF32(float32(math.Inf(1)))
	commit()

	desc("f32: -Inf")
	e.WriteF32(float32(math.Inf(-1)))
	commit()

	// float 64-bit
	desc("f64: 0")
	e.WriteF64(0)
	commit()

	desc("f64: smallest non-zero")
	e.WriteF64(math.SmallestNonzeroFloat64)
	commit()

	desc("f64: MAX")
	e.WriteF64(math.MaxFloat64)
	commit()

	desc("f64: MIN")
	e.WriteF64(-math.MaxFloat64)
	commit()

	desc("f64: NaN")
	e.WriteF64(math.NaN())
	commit()

	desc("f64: +Inf")
	e.WriteF64(math.Inf(1))
	commit()

	desc("f64: -Inf")
	e.WriteF64(math.Inf(-1))
	commit()

	// String
	desc("string: ''")
	e.WriteString("")
	commit()

	desc("string: ' '")
	e.WriteString(" ")
	commit()

	desc("string: '𝐋ṍ𝒓ḝм ℹꝑȿ𝘂м ԁ𝙤ŀ𝖔𝒓 𝘴𝝸ť 𝒂ᵯ𝕖ṯ'")
	e.WriteString("𝐋ṍ𝒓ḝм ℹꝑȿ𝘂м ԁ𝙤ŀ𝖔𝒓 𝘴𝝸ť 𝒂ᵯ𝕖ṯ")
	commit()

	desc("string: a\\u0000b")
	e.WriteString("a\u0000b")
	commit()

	desc("string: F4 80 83 92")
	e.WriteTag(ltv.String, ltv.Size1)
	e.RawWriteByte(4)
	e.RawWrite([]byte{0xF4, 0x80, 0x83, 0x92})
	commit()

	// Struct
	desc("struct: {}")
	e.WriteStructStart()
	e.WriteStructEnd()
	commit()

	desc("struct: {'a': {'b': {'c': 5} } }")
	e.WriteStructStart()
	e.WriteString("a")
	e.WriteStructStart()
	e.WriteString("b")
	e.WriteStructStart()
	e.WriteString("c")
	e.WriteU8(5)
	e.WriteStructEnd()
	e.WriteStructEnd()
	e.WriteStructEnd()
	commit()

	// List
	desc("list: []")
	e.WriteListStart()
	e.WriteListEnd()
	commit()

	desc("list: [[[5]]]")
	e.WriteListStart()
	e.WriteListStart()
	e.WriteListStart()
	e.WriteU8(5)
	e.WriteListEnd()
	e.WriteListEnd()
	e.WriteListEnd()
	commit()

	// Vector boundaries
	desc("u8[]: 0-254")
	vu8 := make([]uint8, 255)
	for i := 0; i < 255; i++ {
		vu8[i] = uint8(i)
	}
	e.WriteVecU8(vu8)
	commit()

	desc("u8[]: 0-255")
	vu8 = make([]uint8, 256)
	for i := 0; i < 256; i++ {
		vu8[i] = uint8(i)
	}
	e.WriteVecU8(vu8)
	commit()

	desc("u32[]: 0-10, unaligned")
	vu32 := make([]uint32, 10)
	for i := 0; i < 10; i++ {
		vu32[i] = uint32(i)
	}
	// Lie to the encoder about the current offset
	e.SetOffset(3)
	e.WriteVecU32(vu32)
	commit()

	desc("i64[]: 0-10, no alignment")
	vi64 := make([]int64, 10)
	for i := 0; i < 10; i++ {
		vi64[i] = int64(i)
	}
	// Lie to the encoder about the current offset
	e.SetOffset(6)
	e.WriteVecI64(vi64)
	commit()

	desc("string 'trail'<NOP><NOP><NOP>")
	e.WriteString("trail")
	e.WriteNop()
	e.WriteNop()
	e.WriteNop()
	commit()

	////////////////////////////////////////////////////////////////////////////////
	// Marshaled Structures
	////////////////////////////////////////////////////////////////////////////////

	// A small vector with alignment padding.
	marshal(struct {
		S string
		I []int32
	}{
		S: "ab",
		I: []int32{1},
	})

	marshal(struct {
		String      string
		NumberOne   int
		NumberTwo   int
		NumberThree int
		NumberFour  int
	}{
		String:      "This is a string",
		NumberOne:   127,
		NumberTwo:   129,
		NumberThree: 257,
		NumberFour:  -1,
	})

	// Integer bounds
	marshal(struct {
		U8  uint8
		U16 uint16
		U32 uint32
		U64 uint64
		I8  int8
		I16 int16
		I32 int32
		I64 int64
	}{
		U8:  math.MaxUint8,
		U16: math.MaxUint16,
		U32: math.MaxUint32,
		U64: math.MaxUint64,
		I8:  math.MinInt8,
		I16: math.MinInt16,
		I32: math.MinInt32,
		I64: math.MinInt64,
	})

	// Arrays
	marshal(struct {
		Bools []bool
		U8s   []uint8
		U16s  []uint16
		U32s  []uint32
		U64s  []uint64
		I8s   []int8
		I16s  []int16
		I32s  []int32
		I64s  []int64
	}{
		Bools: []bool{true, false, false, true},
		U8s:   []uint8{1, 2, 3},
		U16s:  []uint16{1, 2, 3},
		U32s:  []uint32{1, 2, 3},
		U64s:  []uint64{1, 2, 3},
		I8s:   []int8{1, 2, 3},
		I16s:  []int16{1, 2, 3},
		I32s:  []int32{1, 2, 3},
		I64s:  []int64{1, 2, 3},
	})

	// Misc
	marshal(struct {
		Nil  *byte
		Bool bool
		Str  string
		List []any
		Map  map[string]any
	}{
		Nil:  nil,
		Bool: true,
		Str:  "Hello string",
		List: []any{"Bill", "Ted", 2, true},
		Map:  map[string]any{"Bill": 1, "Ted": 2, "Band": "Wyld Stallyns", "Bogus": false},
	})
}

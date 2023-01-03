package testvectors

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"

	ltv "github.com/ThadThompson/ltvgo"
)

//////////////////////////////////////////////////////////////////////////////
// Negative Vectors
//////////////////////////////////////////////////////////////////////////////

func GenerateNegativeVectors(w io.Writer) {

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

	invalidTag := func(typeCode int, sizeCode int) {
		desc(fmt.Sprintf("invalid tag - typeCode: %d, sizeCode: %d", typeCode, sizeCode))
		e.WriteTag(ltv.TypeCode(typeCode), ltv.SizeCode(sizeCode))
		commit()
	}

	// Invalid type/size tag combinations
	for typeCode := 0; typeCode < 16; typeCode++ {

		// Types <= End should only have a 0 size code
		if typeCode <= int(ltv.End) {
			for sizeCode := int(ltv.Size1); sizeCode <= int(ltv.Size8); sizeCode++ {
				invalidTag(typeCode, sizeCode)
			}
		}

		// All size codes > Size8 are invalid
		for sizeCode := int(ltv.Size8) + 1; sizeCode < 16; sizeCode++ {

			// Skip the special cased NOP tag
			if typeCode == int(ltv.Nil) && sizeCode == 0xF {
				continue
			}

			invalidTag(typeCode, sizeCode)
		}
	}

	// Truncated types
	desc("Truncated u8")
	e.WriteTag(ltv.U8, ltv.SizeSingle)
	commit()

	desc("Truncated u16")
	e.WriteTag(ltv.U16, ltv.SizeSingle)
	e.RawWriteByte(0)
	commit()

	desc("Truncated u32")
	e.WriteTag(ltv.U32, ltv.SizeSingle)
	e.RawWriteByte(1)
	e.RawWriteByte(2)
	commit()

	desc("Truncated u64")
	e.WriteTag(ltv.U64, ltv.SizeSingle)
	e.RawWriteByte(1)
	e.RawWriteByte(2)
	e.RawWriteByte(3)
	e.RawWriteByte(4)
	e.RawWriteByte(5)
	e.RawWriteByte(6)
	e.RawWriteByte(7)
	commit()

	desc("Truncated i8")
	e.WriteTag(ltv.I8, ltv.SizeSingle)
	commit()

	desc("Truncated i16")
	e.WriteTag(ltv.I16, ltv.SizeSingle)
	commit()

	desc("Truncated i32")
	e.WriteTag(ltv.I32, ltv.SizeSingle)
	e.RawWriteByte(1)
	e.RawWriteByte(2)
	e.RawWriteByte(3)
	commit()

	desc("Truncated i64")
	e.WriteTag(ltv.I64, ltv.SizeSingle)
	e.RawWriteByte(1)
	e.RawWriteByte(2)
	e.RawWriteByte(3)
	e.RawWriteByte(4)
	e.RawWriteByte(5)
	e.RawWriteByte(6)
	commit()

	desc("Truncated f32")
	e.WriteTag(ltv.F32, ltv.SizeSingle)
	e.RawWriteByte(1)
	commit()

	desc("Truncated f64")
	e.WriteTag(ltv.F64, ltv.SizeSingle)
	e.RawWriteByte(1)
	e.RawWriteByte(2)
	e.RawWriteByte(3)
	e.RawWriteByte(4)
	e.RawWriteByte(5)
	e.RawWriteByte(6)
	commit()

	// Truncated messages
	desc("Truncated []f32 length")
	e.WriteTag(ltv.F32, ltv.Size2)
	e.RawWriteByte(uint8(99))
	commit()

	desc("Truncated []f32")
	e.WriteTag(ltv.F32, ltv.Size1)
	e.RawWriteByte(uint8(4))
	e.RawWriteByte(uint8(0))
	e.RawWriteByte(uint8(0))
	e.RawWriteByte(uint8(0))
	commit()

	desc("Truncated struct {'a': ")
	e.WriteStructStart()
	e.WriteString("a")
	commit()

	desc("Truncated list [1, 2 ")
	e.WriteListStart()
	e.WriteU8(1)
	e.WriteU8(2)
	commit()

	desc("Truncated struct {'a': <nop><nop>")
	e.WriteStructStart()
	e.WriteString("a")
	e.WriteNop()
	e.WriteNop()
	commit()

	// Invalid strings
	desc("Invalid UTF-8 (first byte)")
	e.WriteTag(ltv.String, ltv.Size1)
	e.RawWriteByte(1)
	e.RawWrite([]byte{0xF6})
	commit()

	desc("Invalid UTF-8 (first byte)")
	e.WriteTag(ltv.String, ltv.Size1)
	e.RawWriteByte(2)
	e.RawWrite([]byte{0xC0, 0xAF})
	commit()

	desc("Invalid UTF-8 (second byte)")
	e.WriteTag(ltv.String, ltv.Size1)
	e.RawWriteByte(3)
	e.RawWrite([]byte{0xE0, 0x9F, 0x80})
	commit()

	desc("String (that isn't actually there)")
	e.WriteTag(ltv.String, ltv.Size8)
	e.RawWriteByte(0xFF)
	commit()

	desc("Long string (that isn't actually there)")
	e.WriteTag(ltv.String, ltv.Size8)
	e.RawWriteUint64(0xFFFFFFFFFFFFFFFF)
	commit()

	// Mangled Vector lengths
	desc("short u16[]")
	e.WriteTag(ltv.U16, ltv.Size1)
	e.RawWriteByte(uint8(1))
	e.RawWriteUint16(2000)
	commit()

	desc("long u16[]")
	e.WriteTag(ltv.U16, ltv.Size1)
	e.RawWriteByte(uint8(3))
	e.RawWriteUint16(2000)
	e.RawWriteByte(55)
	commit()

	desc("short []u64")
	e.WriteTag(ltv.U64, ltv.Size1)
	e.RawWriteByte(uint8(15))
	e.RawWriteUint64(5)
	e.RawWriteUint64(6)
	commit()

	desc("long []u64")
	e.WriteTag(ltv.U64, ltv.Size1)
	e.RawWriteByte(uint8(17))
	e.RawWriteUint64(5)
	e.RawWriteUint64(6)
	e.RawWriteUint64(0)
	commit()

	desc("short []f32")
	e.WriteTag(ltv.F32, ltv.Size1)
	e.RawWriteByte(uint8(15))
	e.RawWriteUint64(0)
	e.RawWriteUint64(0)
	commit()

	desc("ludicrous []u64")
	e.WriteTag(ltv.U64, ltv.Size8)
	e.RawWriteUint64(0xFFFFFFFFFFFFFFFF)
	e.RawWriteByte(uint8(99))
	commit()

	// Invalid Lists
	desc("list: ] (hanging list_end)")
	e.WriteListEnd()
	commit()

	desc("list: [[] (unclosed list)")
	e.WriteListStart()
	e.WriteListStart()
	e.WriteListEnd()
	commit()

	desc("list: ['A', 123, [false] (unclosed list)")
	e.WriteListStart()
	e.WriteString("A")
	e.WriteInt(123)
	e.WriteListStart()
	e.WriteBool(false)
	e.WriteListEnd()
	commit()

	// Invalid Structs
	desc("struct: {'a':} (no struct value)")
	e.WriteStructStart()
	e.WriteString("a")
	e.WriteStructEnd()
	commit()

	desc("struct: {nil:5} (nil struct key)")
	e.WriteStructStart()
	e.WriteNil()
	e.WriteInt(5)
	e.WriteStructEnd()
	commit()

	desc("struct: {'':5} (zero length struct key)")
	e.WriteStructStart()
	e.WriteString("")
	e.WriteInt(5)
	e.WriteStructEnd()
	commit()

	desc("struct: {5:'five'} (non-string struct key)")
	e.WriteStructStart()
	e.WriteU8(5)
	e.WriteString("five")
	e.WriteStructEnd()
	commit()

	desc("struct: {'0':['0', <corrupt string>")
	e.WriteStructStart()
	e.WriteString("0")
	e.WriteListStart()
	e.WriteString("0")
	e.WriteTag(ltv.String, ltv.Size1)
	e.RawWriteByte(0x30)
	e.RawWriteByte(0x00)
	commit()
}

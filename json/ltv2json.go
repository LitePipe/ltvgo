package json

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	ltv "github.com/ThadThompson/ltvgo"
)

const indent = "    "

// Streaming LiteVector to JSON transcoder
func Ltv2Json(reader io.Reader, writer io.Writer, prettyPrint bool) error {

	w := bufio.NewWriter(writer)
	s := ltv.NewStreamDecoder(reader)
	var buf [8]byte
	firstPrint := true

	for {

		d, err := s.Next()
		if err != nil {
			w.Flush()
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch d.Role {
		case ltv.RoleStructEnd:
			if prettyPrint {
				w.WriteRune('\n')
				for i := 0; i < d.Depth; i++ {
					w.WriteString(indent)
				}
			}
			w.WriteRune('}')
			firstPrint = false
			continue
		case ltv.RoleListEnd:
			if prettyPrint {
				w.WriteRune('\n')
				for i := 0; i < d.Depth; i++ {
					w.WriteString(indent)
				}
			}
			w.WriteRune(']')
			firstPrint = false
			continue
		}

		if d.Role == ltv.RoleStructValue {
			w.WriteRune(':')
		} else if !d.FirstElement {
			w.WriteRune(',')
		}

		if prettyPrint && d.Role != ltv.RoleStructValue && !firstPrint {
			w.WriteRune('\n')
			for i := 0; i < d.Depth; i++ {
				w.WriteString(indent)
			}
		}

		typeSize := uint64(d.TypeCode.Size())

		switch d.TypeCode {
		case ltv.Nil:
			w.WriteString("null")
		case ltv.Struct:
			w.WriteRune('{')

		case ltv.List:
			w.WriteRune('[')

		case ltv.String:
			w.WriteRune('"')
			n, err := io.Copy(w, io.LimitReader(s, int64(d.Length)))
			if err != nil || n != int64(d.Length) {
				return io.ErrUnexpectedEOF
			}
			w.WriteRune('"')
		}

		if d.TypeCode < ltv.Bool {
			firstPrint = false
			continue
		}

		if d.SizeCode != ltv.SizeSingle {
			w.WriteRune('[')
		}

		// Elements
		for i := 0; i < int(d.Length); i += int(typeSize) {

			n, err := s.Read(buf[0:typeSize])
			if err != nil || n != int(typeSize) {
				return io.ErrUnexpectedEOF
			}

			if i > 0 {
				w.WriteString(", ")
			}

			val := buf[0:typeSize]

			switch d.TypeCode {
			case ltv.Bool:
				if buf[0] == 0 {
					w.WriteString("false")
				} else {
					w.WriteString("true")
				}
			case ltv.U8:
				fmt.Fprint(w, uint8(val[0]))
			case ltv.U16:
				fmt.Fprint(w, binary.LittleEndian.Uint16(val))
			case ltv.U32:
				fmt.Fprint(w, binary.LittleEndian.Uint32(val))
			case ltv.U64:
				fmt.Fprintf(w, "\"%d\"", binary.LittleEndian.Uint64(val))
			case ltv.I8:
				fmt.Fprint(w, int8(val[0]))
			case ltv.I16:
				fmt.Fprint(w, int16(binary.LittleEndian.Uint16(val)))
			case ltv.I32:
				fmt.Fprint(w, int32(binary.LittleEndian.Uint32(val)))
			case ltv.I64:
				fmt.Fprintf(w, "\"%d\"", int64(binary.LittleEndian.Uint64(val)))
			case ltv.F32:
				printFloat(w, math.Float32frombits(binary.LittleEndian.Uint32(val)))
			case ltv.F64:
				printFloat(w, math.Float64frombits(binary.LittleEndian.Uint64(val)))
			}
		}

		if d.SizeCode != ltv.SizeSingle {
			w.WriteRune(']')
		}
		firstPrint = false
	}
}

func printFloat[V float32 | float64](w io.Writer, v V) {
	if math.IsNaN(float64(v)) {
		fmt.Fprintf(w, "\"%s\"", floatNan)
	} else if math.IsInf(float64(v), 1) {
		fmt.Fprintf(w, "\"%s\"", floatPosInf)
	} else if math.IsInf(float64(v), -1) {
		fmt.Fprintf(w, "\"%s\"", floatNegInf)
	} else {
		fmt.Fprintf(w, "%g", v)
	}
}

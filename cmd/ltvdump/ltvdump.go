// A diagnostic utility to dump LiteVector data field by field

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	ltv "github.com/ThadThompson/ltvgo"
)

func abort(msg string) {
	fmt.Println()
	fmt.Print("*** ")
	fmt.Print(msg)
	fmt.Println(" ***")
	os.Exit(1)
}

func ltvDump(reader io.Reader) {

	s := ltv.NewStreamDecoder(reader)
	s.ReturnNops = true
	fmt.Println("TagOffset  Tag   Type   Size   Length  ValueOffset  Value ")
	var buf [8]byte

	for {

		d, err := s.Next()

		if err != io.EOF {
			fmt.Printf(" %08X   ", d.TagOffset)
			fmt.Printf("%02X   %-8s  %d ", d.Tag, d.TypeCode, d.SizeCode)
		}

		if err != nil {
			if err == io.EOF {
				return
			}
			abort(fmt.Sprint(err))
		}

		// Length
		fmt.Printf("%8d     ", d.Length)

		// Value offset
		fmt.Printf("%08X  ", d.ValueOffset)

		if d.Tag == ltv.NopTag {
			fmt.Printf("(nop)")
			fmt.Println()
			continue
		}

		// Single elements
		switch d.TypeCode {
		case ltv.Nil:
			fmt.Printf("(nil)")
		case ltv.Struct:
			fmt.Printf("{")
		case ltv.End:
			if d.Role == ltv.RoleStructEnd {
				fmt.Printf("}")
			} else if d.Role == ltv.RoleListEnd {
				fmt.Printf("]")
			}
		case ltv.List:
			fmt.Printf("[")
		}

		if d.TypeCode <= ltv.End {
			fmt.Println()
			continue
		}

		// String
		if d.TypeCode == ltv.String {
			buf := new(strings.Builder)
			n, err := io.Copy(buf, io.LimitReader(s, int64(d.Length)))
			if err != nil || n != int64(d.Length) {
				abort("Unexpected EOF")
			}
			fmt.Print("\"")
			fmt.Print(buf.String())
			fmt.Println("\"")
			continue
		}

		if d.SizeCode != ltv.SizeSingle {
			fmt.Print("[")
		}

		// Elements
		typeSize := uint64(d.TypeCode.Size())
		for i := 0; i < int(d.Length); i += int(typeSize) {
			n, err := s.Read(buf[0:typeSize])
			if err != nil || n != int(typeSize) {
				abort("unexpected EOF")
			}

			if i > 0 {
				fmt.Print(", ")
			}

			val := buf[0:typeSize]

			switch d.TypeCode {
			case ltv.Bool:
				if buf[0] == 0 {
					fmt.Print("false")
				} else {
					fmt.Print("true")
				}
			case ltv.U8:
				fmt.Print(uint8(val[0]))
			case ltv.U16:
				fmt.Print(binary.LittleEndian.Uint16(val))
			case ltv.U32:
				fmt.Print(binary.LittleEndian.Uint32(val))
			case ltv.U64:
				fmt.Print(binary.LittleEndian.Uint64(val))
			case ltv.I8:
				fmt.Print(int8(val[0]))
			case ltv.I16:
				fmt.Print(int16(binary.LittleEndian.Uint16(val)))
			case ltv.I32:
				fmt.Print(int32(binary.LittleEndian.Uint32(val)))
			case ltv.I64:
				fmt.Print(int64(binary.LittleEndian.Uint64(val)))
			case ltv.F32:
				fmt.Print(math.Float32frombits(binary.LittleEndian.Uint32(val)))
			case ltv.F64:
				fmt.Print(math.Float64frombits(binary.LittleEndian.Uint64(val)))
			}
		}

		if d.SizeCode != ltv.SizeSingle {
			fmt.Print("]")
		}

		fmt.Println()
	}
}

func main() {
	hexEncoded := flag.Bool("x", false, "hex encoded input")
	inputFile := flag.String("i", "", "read input from a file")
	flag.Parse()

	var r io.Reader

	if len(*inputFile) > 0 {
		// Read from file
		fin, err := os.Open(*inputFile)
		if err != nil {
			abort(fmt.Sprintf("unable to open file: %s", err))
		}
		r = fin
	} else if len(flag.Args()) > 0 {
		// Decode from command line
		r = bytes.NewReader([]byte(flag.Arg(0)))
	} else {
		// Read from stdin
		r = os.Stdin
	}

	if *hexEncoded {
		ltvDump(hex.NewDecoder(r))
	} else {
		ltvDump(r)
	}
}

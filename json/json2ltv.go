package json

import (
	"encoding/json"
	"io"
	"math"
	"strconv"

	ltv "github.com/ThadThompson/ltvgo"
)

// Streaming JSON to LiteVector transcoder.
func Json2Ltv(r io.Reader, w io.Writer) error {

	e := ltv.NewEncoder(w)
	dec := json.NewDecoder(r)
	dec.UseNumber()

	var lst *GoldiList

	for {

		// Check for write errors in the last pass
		if e.Werr != nil {
			return e.Werr
		}

		token, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// If we're building a goldilist, then let it try to handle this token
		if lst != nil {
			status := lst.Add(token)
			if status == statusBuilding {
				continue
			} else if status == statusComplete {
				writeGoldiAsVector(lst, e)
				lst = nil
				continue
			} else if status == statusCollapsed {
				writeGoldiAsList(lst, e)
				lst = nil
				// Fallthrough and let the normal token processor handle it.
			}
		}

		switch token := token.(type) {
		case json.Delim:
			if token == json.Delim('{') {
				e.WriteStructStart()
			} else if token == json.Delim('}') {
				e.WriteStructEnd()
			} else if token == json.Delim('[') {
				lst = NewGoldiList()
			} else if token == json.Delim(']') {
				e.WriteListEnd()
			}

		case string:
			// Check our string to see if there's anything we can pull out of it.
			sval := sniffStringValue([]byte(token))

			switch val := sval.(type) {
			case float64:
				e.WriteF64(val)
			case int64:
				e.WriteInt(val)
			case uint64:
				e.WriteUint(val)
			default:
				e.WriteString(token)
			}

		case json.Number:
			// Try integer first
			i64, err := strconv.ParseInt(string(token), 10, 64)
			if err == nil {
				e.WriteInt(i64)
			} else {

				// Then fallback to float
				f64, err := strconv.ParseFloat(string(token), 64)
				if err != nil {
					return err
				}
				e.WriteF64(f64)
			}

		case nil:
			e.WriteNil()
		case bool:
			e.WriteBool(token)
		default:
			panic("Unhandled JSON Token" + token.(string))
		}
	}

	return nil
}

func writeGoldiAsList(lst *GoldiList, e *ltv.Encoder) {
	e.WriteListStart()

	switch data := lst.data.(type) {
	case nil:
	case []bool:
		for _, v := range data {
			e.WriteBool(v)
		}
	case []uint64:
		for _, v := range data {
			e.WriteU64(v)
		}
	case []int64:
		for _, v := range data {
			e.WriteI64(v)
		}
	case []float64:
		for _, v := range data {
			e.WriteF64(v)
		}
	default:
		panic("Unexpected Goldilist data type")
	}
}

// Write a GoldiList as a strongly typed buffer (or buffer struct)
func writeGoldiAsVector(lst *GoldiList, e *ltv.Encoder) {

	switch data := lst.data.(type) {
	case nil:
		// Pass
	case []bool:
		e.WriteVecBool(data)
	case []float64:
		e.WriteVecF64(data)
	case []uint64:
		// Goldilocks the vector type size
		switch {
		case lst.uMax <= math.MaxUint8:
			e.WriteVectorPrefix(ltv.U8, len(data))
			for _, val := range data {
				e.RawWriteByte(byte(val))
			}
		case lst.uMax <= math.MaxUint16:
			e.WriteVectorPrefix(ltv.U16, len(data))
			for _, val := range data {
				e.RawWriteUint16(uint16(val))
			}
		case lst.uMax <= math.MaxUint32:
			e.WriteVectorPrefix(ltv.U32, len(data))
			for _, val := range data {
				e.RawWriteUint32(uint32(val))
			}
		default:
			e.WriteVecU64(data)
		}
	case []int64:
		switch {
		case lst.iMin >= math.MinInt8 && lst.iMax <= math.MaxInt8:
			e.WriteVectorPrefix(ltv.I8, len(data))
			for _, val := range data {
				e.RawWriteByte(byte(val))
			}
		case lst.iMin >= math.MinInt16 && lst.iMax <= math.MaxInt16:
			e.WriteVectorPrefix(ltv.I16, len(data))
			for _, val := range data {
				e.RawWriteUint16(uint16(val))
			}
		case lst.iMin >= math.MinInt32 && lst.iMax <= math.MaxInt32:
			e.WriteVectorPrefix(ltv.I32, len(data))
			for _, val := range data {
				e.RawWriteUint32(uint32(val))
			}
		default:
			e.WriteVecI64(data)
		}

	default:
		panic("Unexpected Goldilist data type")
	}
}

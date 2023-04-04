package ltvgo

import (
	"bytes"
	"math"
	"reflect"
	"testing"
)

func TestRoundString(t *testing.T) {

	v1 := "Testing 1,2,3..."
	enc, err := Marshal(v1)
	if err != nil {
		t.Fatal(err)
	}

	var v2 string
	if err := Unmarshal(enc, &v2); err != nil {
		t.Fatal(err)
	}
	if v1 != v2 {
		t.Fatal("roundtrip mismatch")
	}

	var v3 any
	if err := Unmarshal(enc, &v3); err != nil {
		t.Fatal(err)
	}

	if v1 != v3 {
		t.Fatal("roundtrip mismatch")
	}
}

func TestRoundBytes(t *testing.T) {

	v1 := []byte{0x01, 0x02, 0x03}
	enc, err := Marshal(v1)
	if err != nil {
		t.Fatal(err)
	}

	var v2 []byte
	if err := Unmarshal(enc, &v2); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(v1, v2) {
		t.Fatal("roundtrip mismatch")
	}

	var v3 any
	if err := Unmarshal(enc, &v3); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(v1, v3.([]byte)) {
		t.Fatal("roundtrip mismatch")
	}
}

func TestRoundBool(t *testing.T) {

	v1 := true
	enc, err := Marshal(v1)
	if err != nil {
		t.Fatal(err)
	}

	var v2 bool
	if err := Unmarshal(enc, &v2); err != nil {
		t.Fatal(err)
	}

	if v1 != v2 {
		t.Fatal("roundtrip mismatch")
	}

	var v3 any
	if err := Unmarshal(enc, &v3); err != nil {
		t.Fatal(err)
	}

	if v1 != v2 {
		t.Fatal("roundtrip mismatch")
	}
}

func TestRoundMap(t *testing.T) {

	v1 := make(map[string]any)
	v1["a"] = 55
	v1["b"] = 123.456
	v1["c"] = "map test"

	enc, err := Marshal(v1)
	if err != nil {
		t.Fatal(err)
	}

	v2 := make(map[string]any)
	if err := Unmarshal(enc, &v2); err != nil {
		t.Fatal(err)
	}

	if v1["a"] != v2["a"] {
		t.Fatal("roundtrip mismatch")
	}

	if v1["c"] != v2["c"] {
		t.Fatal("roundtrip mismatch")
	}
}

func TestRoundStruct(t *testing.T) {
	type LtvCheck struct {
		Nil  *byte
		Bool bool
		U8   uint8
		U16  uint16
		U32  uint32
		U64  uint64
		I8   int8
		I16  int16
		I32  int32
		I64  int64
		F32  float32
		F64  float64
		//F64_nan float64
		F64_inf float64
		Str     string
		Bytes   []byte
		Bools   []bool
		I8s     []int8
		I16s    []int16
		I32s    []int32
		I64s    []int64
		U8s     []uint8
		U16s    []uint16
		U32s    []uint32
		U64s    []uint64
		F32s    []float32
		F64s    []float64
		List    []any
		Map     map[string]interface{}
	}

	v1 := LtvCheck{
		Nil:  nil,
		Bool: true,
		U8:   225,
		U16:  50000,
		U32:  math.MaxUint32,
		U64:  math.MaxUint64,
		I8:   -123,
		I16:  1234,
		I32:  -40,
		I64:  -123456,
		F32:  123.456,
		F64:  789.101112,
		//F64_nan: math.NaN(),
		F64_inf: math.Inf(1),
		Str:     "This is a string",
		Bytes:   []byte("These are bytes"),
		Bools:   []bool{true, true, false, false},
		I8s:     []int8{1, 2, 3},
		I16s:    []int16{1, 2, 3},
		I32s:    []int32{1, 2, 3},
		I64s:    []int64{1, 2, 3},
		U8s:     []uint8{1, 2, 3},
		U16s:    []uint16{1, 2, 3},
		U32s:    []uint32{1, 2, 3},
		U64s:    []uint64{1, 2, 3},
		F32s:    []float32{1.1, 2.2, 3.3},
		F64s:    []float64{1.1, 2.2, 3.3},
		List:    []any{"Bob", "the", 7, "builder"},
		Map:     map[string]interface{}{"Bill": "Ted", "Sulu": 7},
	}

	enc, err := Marshal(&v1)
	if err != nil {
		t.Fatal(err)
	}

	//fmt.Println(hex.EncodeToString(enc))

	var v2 LtvCheck
	if err := Unmarshal(enc, &v2); err != nil {
		t.Fatal(err)
	}

	// Full struct check
	// Notice that DeepEqual will fail comparing floating point NaN values
	// because NaN is defined to not equal any other number or itself.
	if !reflect.DeepEqual(v1, v2) {
		t.Fatal("v1 and v2 are not DeepEqual")
	}
}

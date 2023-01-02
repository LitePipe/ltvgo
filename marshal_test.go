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

func TestRoundLtvMap(t *testing.T) {
	v1 := NewLtvStruct()
	v1.Set("z", 55)
	v1.Set("y", "testing")
	v1.Set("a", 123.456)

	enc, err := Marshal(v1)
	if err != nil {
		t.Fatal(err)
	}

	var v2 LtvStruct
	if err := Unmarshal(enc, &v2); err != nil {
		t.Fatal(err)
	}

	if v1.String() != v2.String() {
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

func assertEqual[K comparable](t *testing.T, tag string, a K, b K) {
	if a != b {
		t.Fatal("Comparison failed: " + tag)
	}
}

func assertEqualSlice[K comparable](t *testing.T, tag string, a []K, b []K) {
	if len(a) != len(b) {
		t.Fatal("Comparison failed: " + tag)
	}

	for i, v := range a {
		if v != b[i] {
			t.Fatal("Comparison failed: " + tag)
		}
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
		//F64_inf float64
		Str   string
		Bytes []byte
		Bools []bool
		I8s   []int8
		I16s  []int16
		I32s  []int32
		I64s  []int64
		U8s   []uint8
		U16s  []uint16
		U32s  []uint32
		U64s  []uint64
		F32s  []float32
		F64s  []float64
		List  []any
		Map   map[string]interface{}
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
		//F64_inf: math.Inf(1),
		Str:   "This is a string",
		Bytes: []byte("These are bytes"),
		Bools: []bool{true, true, false, false},
		I8s:   []int8{1, 2, 3},
		I16s:  []int16{1, 2, 3},
		I32s:  []int32{1, 2, 3},
		I64s:  []int64{1, 2, 3},
		U8s:   []uint8{1, 2, 3},
		U16s:  []uint16{1, 2, 3},
		U32s:  []uint32{1, 2, 3},
		U64s:  []uint64{1, 2, 3},
		F32s:  []float32{1.1, 2.2, 3.3},
		F64s:  []float64{1.1, 2.2, 3.3},
		List:  []any{"Bob", "the", 7, "builder"},
		Map:   map[string]interface{}{"Bill": "Ted", "Sulu": 7},
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
	if !reflect.DeepEqual(v1, v2) {
		t.Fatal("v1 and v2 are not DeepEqual")
	}

	assertEqual(t, "Nil", v1.Nil, v2.Nil)
	assertEqual(t, "Bool", v1.Bool, v2.Bool)

	assertEqual(t, "U8", v1.U8, v2.U8)
	assertEqual(t, "U16", v1.U16, v2.U16)
	assertEqual(t, "U32", v1.U32, v2.U32)
	assertEqual(t, "U64", v1.U64, v2.U64)

	assertEqual(t, "I8", v1.I8, v2.I8)
	assertEqual(t, "I16", v1.I16, v2.I16)
	assertEqual(t, "I32", v1.I32, v2.I32)
	assertEqual(t, "I64", v1.I64, v2.I64)

	assertEqual(t, "Str", v1.Str, v2.Str)

	assertEqualSlice(t, "Bytes", v1.Bytes, v2.Bytes)
	assertEqualSlice(t, "Bools", v1.Bools, v2.Bools)

	assertEqualSlice(t, "U8s", v1.U8s, v2.U8s)
	assertEqualSlice(t, "U16s", v1.U16s, v2.U16s)
	assertEqualSlice(t, "U32s", v1.U32s, v2.U32s)
	assertEqualSlice(t, "U64s", v1.U64s, v2.U64s)

	assertEqualSlice(t, "I8s", v1.I8s, v2.I8s)
	assertEqualSlice(t, "I16s", v1.I16s, v2.I16s)
	assertEqualSlice(t, "I32s", v1.I32s, v2.I32s)
	assertEqualSlice(t, "I64s", v1.I64s, v2.I64s)

	for i, val := range v1.List {
		if v2.List[i] != val {
			t.Fatal("List roundtrip failed on element ", i, val)
		}
	}

	if v1.Map["Bill"] != v2.Map["Bill"] {
		t.Fatal("Map roundtrip failed")
	}
}

type selfMarshaller struct {
	myInt  int64
	myBool bool
}

func (m *selfMarshaller) MarshalLTV() ([]byte, error) {
	var dataBuf bytes.Buffer
	e := NewEncoder(&dataBuf)

	e.WriteStructStart()

	if m.myBool {
		e.WriteString("truely")
	} else {
		e.WriteString("falsely")
	}
	e.WriteI64(m.myInt)

	e.WriteStructEnd()

	return dataBuf.Bytes(), nil
}

func (m *selfMarshaller) UnmarshalLTV(buf []byte) error {
	var d Decoder
	d.Init(buf)

	// Just use the normal struct deserializer
	value, err := d.Value()
	if err != nil {
		return err
	}

	s, ok := value.(*LtvStruct)
	if !ok {
		return errExpectedStruct
	}

	if val, ok := s.TryGet("truely"); ok {
		m.myBool = true
		m.myInt = val.(int64)
	} else if val, ok := s.TryGet("falsely"); ok {
		m.myBool = false
		m.myInt = val.(int64)
	} else {
		return errExpectedValue
	}

	return nil
}

func TestRoundMarshalLtv(t *testing.T) {

	type Outer struct {
		I32s  []int32
		Marsh selfMarshaller
		I64s  []int64
	}

	v1 := Outer{
		I32s:  []int32{1, 2, 3},
		Marsh: selfMarshaller{myInt: 123, myBool: true},
		I64s:  []int64{1, 2, 3},
	}

	enc, err := Marshal(&v1)
	if err != nil {
		t.Fatal(err)
	}

	var v2 Outer
	if err := Unmarshal(enc, &v2); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(v1, v2) {
		t.Fatal("v1 & v2 not DeepEqual")
	}
}

package json

import (
	"bytes"
	"math"
	"reflect"
	"testing"

	ltv "github.com/ThadThompson/ltvgo"
)

func TestBasicRoundTrip(t *testing.T) {

	type ColorGroup struct {
		ID     int
		Name   string
		Colors []string
		Sub    struct {
			Variation string
			Approved  bool
		}
	}

	group := ColorGroup{
		ID:     1,
		Name:   "Reds",
		Colors: []string{"Crimson", "Red", "Ruby", "Maroon"},
	}
	group.Sub.Variation = "assorted"
	group.Sub.Approved = false

	b, err := ltv.Marshal(group)
	if err != nil {
		t.Fatal(err)
	}

	ltvBuf := bytes.NewBuffer(b)
	jsBuf := &bytes.Buffer{}
	err = Ltv2Json(ltvBuf, jsBuf, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestExtendedTrip(t *testing.T) {
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

	// Serialize -> LTV
	ltvbin, err := ltv.Marshal(v1)
	if err != nil {
		t.Fatal(err)
	}

	//fmt.Println(hex.EncodeToString(ltvbin))

	// Convert LTV -> JSON
	jsBuf := &bytes.Buffer{}
	err = Ltv2Json(bytes.NewBuffer(ltvbin), jsBuf, true)
	if err != nil {
		t.Fatal(err)
	}

	//fmt.Println(jsBuf.String())

	// Convert JSON -> LTV
	ltvBuf := &bytes.Buffer{}
	err = Json2Ltv(jsBuf, ltvBuf)
	if err != nil {
		t.Fatal(err)
	}

	//fmt.Println(hex.EncodeToString(ltvBuf.Bytes()))

	// LTV -> Deserialize
	var v2 LtvCheck
	if err := ltv.Unmarshal(ltvBuf.Bytes(), &v2); err != nil {
		t.Fatal(err)
	}

	// Full struct check
	if !reflect.DeepEqual(v1, v2) {
		t.Fatal("v1 and v2 are not DeepEqual")
	}
}

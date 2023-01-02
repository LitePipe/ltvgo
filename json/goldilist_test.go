package json

import (
	"encoding/json"
	"strings"
	"testing"
)

func Assert(t *testing.T, cond bool, msg string) {
	if !cond {
		t.Fatal(msg)
	}
}

func GoldiParse(js string) *GoldiList {

	dec := json.NewDecoder(strings.NewReader(js))
	dec.UseNumber()
	lst := NewGoldiList()

	// Eat the first '['
	dec.Token()
	for token, err := dec.Token(); err == nil; token, err = dec.Token() {
		lst.Add(token)
	}

	return lst
}

func TestGoldiUint(t *testing.T) {
	lst := GoldiParse(`[1, 2, 3]`)
	Assert(t, lst.status == statusComplete, "Expected list completion")
	Assert(t, lst.kind == uintKind, "Expected uint list")
	Assert(t, lst.uMax == 3, "Incorrect max integer")
}

func TestGoldiInt(t *testing.T) {
	lst := GoldiParse(`[-1, 2, 3]`)
	Assert(t, lst.status == statusComplete, "Expected list completion")
	Assert(t, lst.kind == intKind, "Expected int list")
	Assert(t, lst.iMin == -1, "Incorrect min integer")
	Assert(t, lst.iMax == 3, "Incorrect max integer")
}

func TestGoldiIntMasked(t *testing.T) {

	// Middle null
	lst := GoldiParse(`[-1, null, 3]`)
	Assert(t, lst.status == statusComplete, "Expected list completion")
	Assert(t, lst.kind == intKind, "Expected int list")
	Assert(t, lst.iMin == -1, "Incorrect min integer")
	Assert(t, lst.iMax == 3, "Incorrect max integer")
	Assert(t, lst.mask != nil, "Expected mask to be set")
	Assert(t, len(lst.mask) == lst.length, "Expected mask length to match data length")

	// Leading null
	lst = GoldiParse(`[null, -1, 3]`)
	Assert(t, lst.kind == intKind, "Expected int list")
	Assert(t, lst.status == statusComplete, "Expected list completion")
	Assert(t, lst.iMin == -1, "Incorrect min integer")
	Assert(t, lst.iMax == 3, "Incorrect max integer")
	Assert(t, lst.mask != nil, "Expected mask to be set")
	Assert(t, len(lst.mask) == lst.length, "Expected mask length to match data length")
}

func TestGoldiFloat(t *testing.T) {
	lst := GoldiParse(`[1.0, 2.1, 3.2]`)
	Assert(t, lst.status == statusComplete, "Expected list completion")
	Assert(t, lst.kind == floatKind, "Expected float list")
}

func TestGoldiIntToFloat(t *testing.T) {
	lst := GoldiParse(`[1, 2, 3.2]`)
	Assert(t, lst.status == statusComplete, "Expected list completion")
	Assert(t, lst.kind == floatKind, "Expected float list")
}

func TestGoldiIntToFloatMasked(t *testing.T) {
	lst := GoldiParse(`[1, 2, null, 3.2]`)
	Assert(t, lst.status == statusComplete, "Expected list completion")
	Assert(t, lst.kind == floatKind, "Expected float list")
	Assert(t, lst.mask != nil, "Expected mask to be set")
	Assert(t, len(lst.mask) == lst.length, "Expected mask length to match data length")
}

func TestGoldiBool(t *testing.T) {
	lst := GoldiParse(`[true, false, true]`)
	Assert(t, lst.status == statusComplete, "Expected list completion")
	Assert(t, lst.kind == boolKind, "Expected bool list")
}

// TODO: Advanced array support

// func TestGoldiShapes(t *testing.T) {
// 	// 2D integer array
// 	lst := GoldiParse(`[[1,2], [3,4], [5,6]]`)
// 	Assert(t, lst.status == statusComplete, "Expected list completion")
// 	Assert(t, lst.kind == uintKind, "Expected unsigned integer list")
// 	Assert(t, lst.uMax == 6, "Incorrect max integer")

// 	Assert(t, lst.shape != nil, "Expected shaped data")
// 	Assert(t, lst.shape[0] == 3, "Expected lst.shape[0] == 3")
// 	Assert(t, lst.shape[1] == 2, "Expected lst.shape[1] == 2")

// 	// 3D float array
// 	js := `[[[0, 1, 2, 3], [4, 5, 6, 7], [8, 9, 10, 11]],
//       	   [[11, 10, 9, 8], [7, 6, 5, 4], [3, 2, 1, 0.01]]]`

// 	lst = GoldiParse(js)

// 	Assert(t, lst.status == statusComplete, "Expected list completion")
// 	Assert(t, lst.kind == floatKind, "Expected float list")

// 	Assert(t, lst.shape != nil, "Expected shaped data")
// 	Assert(t, lst.shape[0] == 2, "Expected lst.shape[0] == 2")
// 	Assert(t, lst.shape[1] == 3, "Expected lst.shape[1] == 3")
// 	Assert(t, lst.shape[2] == 4, "Expected lst.shape[2] == 4")
// }

// func TestGoldiStress(t *testing.T) {

// 	js := `[[[null, null, null, null], [4, "NaN", 6, 7], [8, 9, 10, 11]],
// 		   [[11, 10, "-Infinity", 8], [7, 6, 5, null], [3, 2, 1, 0.01]]]`

// 	lst := GoldiParse(js)
// 	data := lst.data.([]float64)

// 	// Check type
// 	Assert(t, lst.status == statusComplete, "Expected list completion")
// 	Assert(t, lst.kind == floatKind, "Expected float list")

// 	// Spot check values
// 	Assert(t, math.IsNaN(data[5]), "Expected NaN at lst.data[5]")
// 	Assert(t, data[6] == 6, "Expected lst.data[6] == 6")
// 	Assert(t, math.IsInf(data[14], -1), "Expected lst.f64data[14] to be -Infinity")

// 	// Check shape
// 	Assert(t, lst.shape != nil, "Expected shaped data")
// 	Assert(t, lst.shape[0] == 2, "Expected lst.shape[0] == 2")
// 	Assert(t, lst.shape[1] == 3, "Expected lst.shape[1] == 3")
// 	Assert(t, lst.shape[2] == 4, "Expected lst.shape[2] == 4")

// 	// Check mask
// 	Assert(t, lst.mask != nil, "Expected mask to be set")
// 	Assert(t, len(lst.mask) == len(data), "Expected mask length to match data length")
// 	Assert(t, lst.mask[0] == true, "Unexpected mask true value")
// 	Assert(t, lst.mask[4] == false, "Unexpected mask false value")
// }

// func TestGoldiFailures(t *testing.T) {
// 	Assert(t, GoldiParse(`[1, 2, false]`).status == statusCollapsed, "Expected int/bool mismatch collapse")
// 	Assert(t, GoldiParse(`[[1,2],[1],[3,4]]`).status == statusCollapsed, "Expected shape mismatch collapse")
// }

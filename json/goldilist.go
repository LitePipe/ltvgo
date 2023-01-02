package json

import (
	"encoding/json"
	"math"
	"strconv"
)

type goldiListStatus int

const (
	statusBuilding goldiListStatus = iota
	statusComplete
	statusCollapsed
)

type goldiKindCompatibility int

const (
	kindCompatible goldiKindCompatibility = iota
	kindUpgrade
	kindIncompatible
)

// Min/max integers representable in JSON/JavaScript
// See: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Number/MAX_SAFE_INTEGER
const jsMaxSafeInt = 9007199254740991
const jsMinSafeInt = -9007199254740991

type goldiKind uint

const (
	noneKind goldiKind = iota
	boolKind
	uintKind
	intKind
	floatKind
)

// A Goldilocks List
// Given a JSON list (or multidimensional list) it will process it while attempting to find a set of
// LiteVector parameters that can fit it as a vector.
type GoldiList struct {
	data any

	kind        goldiKind
	shape       []int64
	mask        []bool
	count_stack []int64
	count       int64

	length int
	iMin   int64
	iMax   int64
	uMax   uint64

	status goldiListStatus
}

func NewGoldiList() *GoldiList {
	return &GoldiList{
		data:        nil,
		shape:       make([]int64, 1),
		count_stack: make([]int64, 0),
		count:       0,
		length:      0,
		kind:        noneKind,
		iMin:        math.MaxInt64,
		iMax:        math.MinInt64,
		uMax:        0,
	}
}

func (l *GoldiList) checkCompatibility(k goldiKind) goldiKindCompatibility {

	// If no kind has been set, then we're compatible
	if l.kind == noneKind {
		return kindUpgrade
	}

	// Always compatible with the same type
	if l.kind == k {
		return kindCompatible
	}

	// Check uint64 upgrades
	if l.kind == uintKind {

		// int64
		if k == intKind && l.uMax < math.MaxInt64 {
			return kindUpgrade
		}

		// float64
		if k == floatKind && l.uMax <= jsMaxSafeInt {
			return kindUpgrade
		}
	}

	// Check int64 upgrades
	if l.kind == intKind {

		// float64
		if k == floatKind && l.iMax <= jsMaxSafeInt && l.iMin >= jsMinSafeInt {
			return kindUpgrade
		}
	}

	return kindIncompatible
}

func (l *GoldiList) appendDelimiter(token json.Delim) {

	// Embedded structures break array candidacy
	if token == json.Delim('{') || token == json.Delim('}') {
		l.status = statusCollapsed
		return
	}

	// Possible multidimensional array
	if token == json.Delim('[') {

		l.status = statusCollapsed
		return

		// TODO: When implementing advanced vector support in the standard library,
		// remove the above early return, and allow this process to continue.

		l.count_stack = append(l.count_stack, l.count)
		l.count = 0

		// Shape stays one level longer than the count stack
		if len(l.count_stack) == len(l.shape) {
			l.shape = append(l.shape, 0)
		}

		return
	}

	if token == json.Delim(']') {

		levelCount := l.count - 1

		// If we make it to the bottom of the stack, then we're done.
		if len(l.count_stack) == 0 {

			// The first dimension of the shape is the count at this level
			l.shape[0] = levelCount
			l.status = statusComplete
			return
		}

		shapeIdx := len(l.count_stack)

		// Pop the count stack
		l.count = l.count_stack[len(l.count_stack)-1]
		l.count_stack = l.count_stack[:len(l.count_stack)-1]

		if l.shape[shapeIdx] == 0 {
			// If this is the first time we've encountered a close tag at this level, then record the count.
			l.shape[shapeIdx] = levelCount
		} else if l.shape[shapeIdx] != levelCount {

			// If we encounter a close tag at the same level but we've seen a different count, then we have a
			// ragged array - and should collapse the list.
			l.status = statusCollapsed
		}

		return
	}
}

func (l *GoldiList) appendUint64(value uint64) {

	// Check if this is already an int64 list
	if l.kind == intKind {

		// Check range
		if value > math.MaxInt64 {
			l.status = statusCollapsed
			return
		}

		// Forward to the int processor
		l.appendInt64(int64(value))
		return
	}

	// Check if this is already a float64 list
	if l.kind == floatKind {
		if value > jsMaxSafeInt {
			l.status = statusCollapsed
			return
		}

		l.appendFloat64(float64(value))
		return
	}

	// Check compatibility
	comp := l.checkCompatibility(uintKind)
	if comp == kindIncompatible {
		l.status = statusCollapsed
		return
	}

	if comp == kindUpgrade {
		// The only upgrade for this type is from null
		l.data = make([]uint64, l.length)
	}

	// Update bounds
	if value > l.uMax {
		l.uMax = value
	}

	l.data = append(l.data.([]uint64), value)
	l.length++
	l.kind = uintKind

	// Update the mask if present
	if l.mask != nil {
		l.mask = append(l.mask, false)
	}
}

func (l *GoldiList) appendInt64(value int64) {

	// Check if this is already a float64 list
	if l.kind == floatKind {
		if value < jsMinSafeInt || value > jsMaxSafeInt {
			l.status = statusCollapsed
			return
		}

		l.appendFloat64(float64(value))
		return
	}

	comp := l.checkCompatibility(intKind)
	if comp == kindIncompatible {
		l.status = statusCollapsed
		return
	}

	if comp == kindUpgrade {
		data := make([]int64, l.length)

		// Transform (uint64 upgrades to int64)
		if l.data != nil {
			for i, v := range l.data.([]uint64) {
				d := int64(v)
				data[i] = d

				if d < l.iMin {
					l.iMin = d
				}

				if d > l.iMax {
					l.iMax = d
				}
			}
		}

		l.data = data
	}

	// Update integer bounds
	if value < l.iMin {
		l.iMin = value
	}

	if value > l.iMax {
		l.iMax = value
	}

	l.data = append(l.data.([]int64), value)
	l.length++
	l.kind = intKind

	// Update the mask if present
	if l.mask != nil {
		l.mask = append(l.mask, false)
	}
}

func (l *GoldiList) appendFloat64(value float64) {

	comp := l.checkCompatibility(floatKind)

	if comp == kindIncompatible {
		l.status = statusCollapsed
		return
	}

	if comp == kindUpgrade {

		oldData := l.data
		newData := make([]float64, l.length)

		// Only uint64 and int64 upgrade to float
		if l.length > 0 {
			switch oldData := oldData.(type) {
			case []uint64:
				for i, v := range oldData {
					newData[i] = float64(v)
				}
			case []int64:
				for i, v := range oldData {
					newData[i] = float64(v)
				}
			}
		}

		l.data = newData
	}

	l.data = append(l.data.([]float64), value)
	l.length++
	l.kind = floatKind

	// Update the mask if present
	if l.mask != nil {
		l.mask = append(l.mask, false)
	}
}

func (l *GoldiList) appendNil() {

	// Null values turn into a mask
	if l.mask == nil {
		l.mask = make([]bool, l.length)
	}
	l.mask = append(l.mask, true)

	// Append a corresponding placeholder value to the data array.
	switch l.kind {
	case noneKind:
	case boolKind:
		l.data = append(l.data.([]bool), false)
	case uintKind:
		l.data = append(l.data.([]uint64), 0)
	case intKind:
		l.data = append(l.data.([]int64), 0)
	case floatKind:
		l.data = append(l.data.([]float64), 0)
	}
	l.length++
}

func (l *GoldiList) appendBool(value bool) {

	comp := l.checkCompatibility(boolKind)
	if comp == kindIncompatible {
		l.status = statusCollapsed
		return
	}

	if comp == kindUpgrade {
		// The only upgrade for this type is from null
		l.data = make([]bool, l.length)
	}

	l.data = append(l.data.([]bool), value)
	l.length++
	l.kind = boolKind

	// Update the mask if present
	if l.mask != nil {
		l.mask = append(l.mask, false)
	}
}

// Append a number type to the list
func (l *GoldiList) appendNumber(token json.Number) {

	// Try uint64
	u64, err := strconv.ParseUint(string(token), 10, 64)
	if err == nil {
		l.appendUint64(u64)
		return
	}

	// Try int64
	i64, err := strconv.ParseInt(string(token), 10, 64)
	if err == nil {
		l.appendInt64(i64)
		return
	}

	// Try float64
	f64, err := strconv.ParseFloat(string(token), 64)
	if err == nil {
		l.appendFloat64(f64)
		return
	}

	panic("Unable to parse  json.Number")
}

// Add a token to the list.
// Returns true if the list consumed the token and maintains array candidacy.
// False if the added token would coerce it into a normal list (the token is not added).
func (l *GoldiList) Add(token json.Token) goldiListStatus {

	if l.status != statusBuilding {
		return l.status
	}

	l.count++

	switch token := token.(type) {
	case json.Delim:
		l.appendDelimiter(token)
		return l.status

	case json.Number:
		l.appendNumber(token)
		return l.status
	case string:
		val := sniffStringValue([]byte(token))
		switch val := val.(type) {
		case float64:
			l.appendFloat64(val)
		case int64:
			l.appendInt64(val)
		case uint64:
			l.appendUint64(val)
		default:
			// We can't handle strings in compact arrays.
			l.status = statusCollapsed
		}
		return l.status

	case bool:
		l.appendBool(token)
		return l.status

	case nil:
		l.appendNil()
		return l.status

	default:
		panic("Unrecognized JSON Token" + token.(string))
	}
}

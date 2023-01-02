package json

import (
	"bytes"
	"encoding/base64"
	"math"
	"strconv"
)

var floatNan = []byte("NaN")
var floatPosInf = []byte("Infinity")
var floatNegInf = []byte("-Infinity")

// Try parsing a byte string as a float64 NaN/Infinity/-Infinity value.
func tryDecodeSpecialFloat(b []byte) (float64, bool) {

	if bytes.Equal(b, floatNan) {
		return math.NaN(), true
	}

	if bytes.Equal(b, floatPosInf) {
		return math.Inf(1), true
	}

	if bytes.Equal(b, floatNegInf) {
		return math.Inf(-1), true
	}

	return 0, false
}

// Try parsing a byte string as an integer, which may lie
// outside JSON/JavaScript's Number range
// Examples: "-9223372036854775808"
func tryDecodeInt64(b []byte) (int64, bool) {
	val, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

// Try parsing a byte string as an unsigned integer, which may lie
// outside JSON/JavaScript's Number range
// Examples: "18446744073709551615"
func tryDecodeUint64(b []byte) (uint64, bool) {
	val, err := strconv.ParseUint(string(b), 10, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

// Try to decode a string from a base64 binary value
func tryDecodeBase64(b []byte) ([]byte, bool) {

	// Don't try anything under 32 bytes
	// or not a multiple of 4
	if len(b) < 32 || len(b)%4 != 0 {
		return nil, false
	}

	// As a fast first pass check. run a quick pass over a few characters
	// looking for whitespace (which we would expect to appear in text).
	for c := range b[:32] {
		if c == ' ' {
			return nil, false
		}
	}

	// So far so good, make an effort to decode it.
	encoding := base64.StdEncoding.Strict()
	buf := make([]byte, encoding.DecodedLen(len(b)))
	n, err := base64.StdEncoding.Decode(buf, b)

	if err != nil {
		return nil, false
	}

	return buf[0:n], true
}

// Decode a JSON string value, while trying to sniff for other data types that have been stringified.
func sniffStringValue(b []byte) any {

	if f, ok := tryDecodeSpecialFloat(b); ok {
		return f
	}

	if i64, ok := tryDecodeInt64(b); ok {
		return i64
	}

	if u64, ok := tryDecodeUint64(b); ok {
		return u64
	}

	if bin, ok := tryDecodeBase64(b); ok {
		return bin
	}

	return string(b)
}

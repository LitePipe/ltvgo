package json

import (
	"math"
	"testing"
)

func TestSpecialFloat(t *testing.T) {

	// Success
	v, ok := tryDecodeSpecialFloat(floatNan)
	if !ok || !math.IsNaN(v) {
		t.Error("Expected tryDecodeComplex return NaN")
	}

	v, ok = tryDecodeSpecialFloat(floatPosInf)
	if !ok || !math.IsInf(v, 1) {
		t.Error("Expected tryDecodeComplex return Infinity")
	}

	v, ok = tryDecodeSpecialFloat(floatNegInf)
	if !ok || !math.IsInf(v, -1) {
		t.Error("Expected tryDecodeComplex return -Infinity")
	}
}

func TestBase64Detection(t *testing.T) {

	v := sniffStringValue([]byte("VHJ5IG5vdC4gRG8sIG9yIGRvIG5vdC4gVGhlcmUgaXMgbm8gdHJ5LiBUaGlzIGlzIHRoZSB3YXku"))
	if _, ok := v.([]byte); !ok {
		t.Error("Expected decodeStringValue to decode byte[]")
	}

	v = sniffStringValue([]byte("Just a regular string"))
	if _, ok := v.(string); !ok {
		t.Error("Expected decodeStringValue to decode string")
	}
}

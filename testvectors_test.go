package ltvgo

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"io"
	"os"
	"testing"
)

// Process test vectors.
// If 'valid' is true, they should parse correctly.
// If 'valid' is false, they should be flagged as invalid.
func processTestVectors(t *testing.T, r io.Reader, valid bool) {

	s := bufio.NewScanner(r)

	testNumber := 0
	for s.Scan() {

		testNumber++
		testDesc := s.Text()

		s.Scan()

		// Could use testNumber to scope/gate which test we're working on here.

		data, err := hex.DecodeString(s.Text())
		if err != nil {
			t.Fatal(err)
		}

		// Check validation
		if Valid(data) != valid {
			t.Fatalf("Valid did not correctly validate for test vector %d: '%s'", testNumber, testDesc)
		}

		// Check single value decoder
		var d Decoder
		d.Init(data)
		_, err = d.Value()
		if valid && err != nil {
			t.Fatalf("Decoder failed for test vector %d '%s' : %s", testNumber, testDesc, err)
		} else if !valid && err == nil {
			t.Fatalf("Decoder incorrectly decoded test vector %d '%s'", testNumber, testDesc)
		}

		// Check marshaling code
		var val any
		err = Unmarshal(data, &val)
		if valid && err != nil {
			t.Fatalf("Unmarshal failed for test vector %d '%s' : %s", testNumber, testDesc, err)
		} else if !valid && err == nil {
			t.Fatalf("Unmarshal incorrectly decoded test vector %d '%s'", testNumber, testDesc)
		}
	}

}

func TestPositiveVectors(t *testing.T) {
	data, err := os.ReadFile("testvectors/litevectors_positive.txt")
	if err != nil {
		t.Fatal(err)
	}
	buf := bytes.NewBuffer(data)

	// buf := &bytes.Buffer{}
	// tv.GeneratePositiveVectors(buf)

	processTestVectors(t, buf, true)
}

func TestNegativeVectors(t *testing.T) {
	data, err := os.ReadFile("testvectors/litevectors_negative.txt")
	if err != nil {
		t.Fatal(err)
	}
	buf := bytes.NewBuffer(data)

	//buf := &bytes.Buffer{}
	//tv.GenerateNegativeVectors(buf)

	processTestVectors(t, buf, false)
}

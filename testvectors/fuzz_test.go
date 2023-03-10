package testvectors

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"io"
	"testing"

	ltv "github.com/ThadThompson/ltvgo"
)

func loadTestVectorForFuzzing(f *testing.F, r io.Reader) {

	s := bufio.NewScanner(r)
	for s.Scan() {
		s.Scan()

		data, err := hex.DecodeString(s.Text())
		if err != nil {
			f.Fatal(err)
		}

		f.Add(data)
	}
}
func FuzzDecode(f *testing.F) {

	// Load test vectors as fuzzing primers
	buf := &bytes.Buffer{}
	GeneratePositiveVectors(buf)
	loadTestVectorForFuzzing(f, buf)

	GenerateNegativeVectors(buf)
	loadTestVectorForFuzzing(f, buf)

	f.Fuzz(func(t *testing.T, b []byte) {

		// Validation function
		ltv.Valid(b)

		// Single value decoder
		d := ltv.NewDecoder(b)
		d.Value()

		// Unmarshal code
		var val any
		ltv.Unmarshal(b, &val)
	})

}

package testvectors

import (
	"os"
	"testing"
)

// A "unit test" with the side effect of generating and writing
func Test_GenerateVectors(t *testing.T) {

	f, err := os.Create("litevectors_positive.txt")
	if err != nil {
		t.Fatal(err)
	}
	GeneratePositiveVectors(f)
	f.Close()

	f, err = os.Create("litevectors_negative.txt")
	if err != nil {
		t.Fatal(err)
	}
	GenerateNegativeVectors(f)
	f.Close()
}

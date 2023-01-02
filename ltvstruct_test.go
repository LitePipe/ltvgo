package ltvgo

import (
	"fmt"
	"testing"
)

func TestLtvStruct(t *testing.T) {
	m := NewLtvStruct()

	var keys []string

	// Generate keys and insert
	for i := 0; i < 26; i++ {
		k := fmt.Sprintf("map_key_%d", i)
		keys = append(keys, k)
		m.Set(k, i)
	}

	// Check
	for i := 0; i < len(m.keys); i++ {
		if *m.keys[i] != keys[i] {
			t.Fatal("Key order mismatch")
		}
	}
}

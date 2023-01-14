package ltvgo

import "testing"

func TestEncoder(t *testing.T) {
	e := NewEncoder()
	e.WriteU8(123)

	buf := e.Bytes()
	if len(buf) != 2 {
		t.Fatal("Unexpected buf len: ", len(buf))
	}

	if buf[0] != 0x60 {
		t.Fatal("Unexpected tag: ", buf[0])
	}

	if buf[1] != 123 {
		t.Fatal("Unexpected value: ", buf[1])
	}
}

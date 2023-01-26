package ltvgo

import (
	"testing"
)

func TestEncoderSimple(t *testing.T) {
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

func TestEncoderReset(t *testing.T) {
	e := NewEncoder()
	e.WriteString("12345678")

	s1 := append([]byte{}, e.Bytes()...)
	if len(s1) != 10 {
		t.Fatal("Unexpected buf len: ", len(s1))
	}

	e.Reset()
	e.WriteString("999")
	s2 := e.Bytes()

	if len(s2) != 5 {
		t.Fatal("Unexpected buf len: ", len(s2))
	}
}

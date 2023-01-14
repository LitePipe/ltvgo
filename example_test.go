package ltvgo

import (
	"encoding/hex"
	"fmt"
)

func ExampleMarshal() {
	type ColorGroup struct {
		ID     int
		Name   string
		Colors []string
	}
	group := ColorGroup{
		ID:     1,
		Name:   "Reds",
		Colors: []string{"Crimson", "Red", "Ruby", "Maroon"},
	}

	b, err := Marshal(group)
	if err != nil {
		fmt.Println("error:", err)
	}

	fmt.Println(hex.EncodeToString(b))

	// Output:
	// 1041024944a00141044e616d654104526564734106436f6c6f72732041074372696d736f6e410352656441045275627941064d61726f6f6e3030
}

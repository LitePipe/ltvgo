// Utility that converts JSON to it's LiteVector representation.
package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"

	ltvjs "github.com/ThadThompson/ltvgo/json"
)

func main() {
	hexEncoded := flag.Bool("x", false, "hex encoded input")
	inputFile := flag.String("i", "", "read input from file")
	outputFile := flag.String("o", "", "write output to file")
	flag.Parse()

	var r io.Reader
	var w io.Writer

	if len(*inputFile) > 0 {
		// Read from file
		fin, err := os.Open(*inputFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to open input file: ", err)
			os.Exit(1)
		}
		r = fin
	} else if len(flag.Args()) > 0 {
		// Decode from command line
		r = bytes.NewReader([]byte(flag.Arg(0)))
	} else {
		// Read from stdin
		r = os.Stdin
	}

	if len(*outputFile) > 0 {
		// Write to file
		fout, err := os.Create(*outputFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to create output file: ", err)
			os.Exit(1)
		}
		w = fout
	} else {
		// Write to standard out
		w = os.Stdout
	}

	if *hexEncoded {
		ltvjs.Json2Ltv(r, hex.NewEncoder(w))
	} else {
		ltvjs.Json2Ltv(r, w)
	}
}

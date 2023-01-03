// Utility to dump struct keys to a dictionary file for use in compression

package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	ltv "github.com/ThadThompson/ltvgo"
)

// Pull the set of deduplicated struct keys from a LiteVector stream.
func ltvKeys(r io.Reader) (map[string]struct{}, error) {

	s := ltv.NewStreamDecoder(r)
	keySet := make(map[string]struct{})

	strBuilder := new(strings.Builder)

	for {

		d, err := s.Next()
		if err != nil {
			if err == io.EOF {
				return keySet, nil
			}
			return nil, err
		}

		if d.Role == ltv.RoleStructKey {

			// Read and set the string
			strBuilder.Reset()
			_, err := io.Copy(strBuilder, io.LimitReader(s, int64(d.Length)))
			if err != nil {
				return nil, err
			}
			keySet[strBuilder.String()] = struct{}{}
		} else {
			s.SkipValue(d)
		}
	}
}

func ltv2Dict(r io.Reader, w io.Writer) error {
	keySet, err := ltvKeys(r)
	if err != nil {
		return err
	}

	// Write the keys as full LiteVector string elements
	e := ltv.NewEncoder(w)
	for key, _ := range keySet {
		e.WriteString(key)
	}

	return e.Werr
}

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

	var err error
	if *hexEncoded {
		err = ltv2Dict(hex.NewDecoder(r), w)
	} else {
		err = ltv2Dict(r, w)
	}

	if err != nil {
		fmt.Fprint(os.Stderr, err)
	}
}

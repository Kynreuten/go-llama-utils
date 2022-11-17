package environment

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

type EnvironmentTranslationReader struct {
	env    *map[string]string
	reader io.Reader

	tokenScanner bufio.Scanner
	isScanReady  bool

	buffOut bytes.Buffer
}

func (etr *EnvironmentTranslationReader) Initialize(envLookup *map[string]string, r io.Reader) {
	etr.env = envLookup
	etr.reader = bufio.NewReader(r)
	etr.tokenScanner = *bufio.NewScanner(etr.reader)

	etr.isScanReady = true
}

func (etr *EnvironmentTranslationReader) ReadFromScanner(p []byte) (n int, err error) {
	if !etr.isScanReady {
		return 0, errors.New("must be initialized")
	}

	var scanDone bool = false
	for !scanDone && etr.buffOut.Len() < len(p) {
		// Scan for the next line of data
		if scanDone = etr.tokenScanner.Scan(); scanDone && etr.tokenScanner.Err() != nil {
			return 0, etr.tokenScanner.Err()
		}

		// Replace any variables in the line that we can
		if translated, done, missing := ExpandVarString(etr.tokenScanner.Text(), etr.env); !done {
			return 0, fmt.Errorf("variables were referenced, but not defined: \n%v", missing)
		} else {
			if _, err := etr.buffOut.WriteString(translated); err != nil {
				return 0, err
			}
		}
	}

	// Have them read from out output buffer
	return etr.buffOut.Read(p)
}

func (etr *EnvironmentTranslationReader) Read(p []byte) (n int, err error) {
	return etr.ReadFromScanner(p)
}
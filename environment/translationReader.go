package environment

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

type EnvironmentTranslationReader struct {
	env    *VariableMap
	reader io.Reader

	tokenScanner bufio.Scanner
	isScanReady  bool

	buffOut bytes.Buffer
}

func NewTranslator(envLookup *VariableMap, r io.Reader) (translator *EnvironmentTranslationReader) {
	translator = &EnvironmentTranslationReader{}
	translator.env = envLookup
	translator.reader = bufio.NewReader(r)
	translator.tokenScanner = *bufio.NewScanner(translator.reader)

	translator.isScanReady = true
	return translator
}

func (etr *EnvironmentTranslationReader) readFromScanner(p []byte) (n int, err error) {
	if !etr.isScanReady {
		return 0, errors.New("must be initialized")
	}

	var hasMore bool = true
	for hasMore && etr.buffOut.Len() < len(p) {
		// Scan for the next line of data
		hasMore = etr.tokenScanner.Scan()

		//NOTE: May have read some bytes. We're not going to use them if there's a non-EOF error however?
		if !hasMore && etr.tokenScanner.Err() != nil {
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
	return etr.readFromScanner(p)
}

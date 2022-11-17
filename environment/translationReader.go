package environment

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
)

type EnvironmentTranslationReader struct {
	Env *map[string]string
	R   io.Reader

	buff         bytes.Buffer
	strBuilder   strings.Builder
	buffer       []byte
	bufferOffset int

	tokenScanner bufio.Scanner
	isScanReady  bool

	// processedQ deque.Deque[string]
	buffOut bytes.Buffer
}

func (etr *EnvironmentTranslationReader) processString(target string) (translated string, remaining string, err error) {
	// Attempt to translate all variables
	if translated, done, missing := ExpandVarString(target, etr.Env); !done {
		return "", "", fmt.Errorf("variables were referenced, but not defined: \n%v", missing)
	} else {
		// Find anything at the end of the string that could be the start of a new variable. If found then that needs to stay in the buffer for the next read?
		if idx, err := FindEndingPartialIndex(translated); err != nil {
			return "", "", err
		} else if idx >= 0 {
			return translated[0:idx], translated[idx:], nil
		}
		return translated, "", nil
	}
}

func (etr *EnvironmentTranslationReader) processReadBuffer() (err error) {
	line, strErr := etr.buff.ReadString('\n')
	for len(line) > 0 {
		etr.strBuilder.WriteString(line)

		if strErr != nil && strErr != io.EOF {
			//TODO: Handle error?
			return strErr
		}
		if processed, remaining, procErr := etr.processString(etr.strBuilder.String()); procErr != nil {
			// Failed to read it. Get out
			//TODO: Are there situations like this that are recoverable?
			return procErr
		} else {
			// Track the processed line
			if len(remaining) > 0 {
				//TODO: Handle this being EOF?
				etr.strBuilder.WriteString(remaining)
			} else {
				etr.strBuilder.Reset()
			}

			// Write to our buffer for output
			etr.buffOut.WriteString(processed)

			if strErr == io.EOF {
				// Hit the end of the data. We're done
				return nil
			}
			// Read the next line
			line, strErr = etr.buff.ReadString('\n')
		}
	}

	//TODO: Finish processing?
	return nil
}

func (etr *EnvironmentTranslationReader) Initialize(r io.Reader) (err error) {
	etr.R = bufio.NewReader(r)
	etr.tokenScanner = *bufio.NewScanner(etr.R)

	etr.isScanReady = true

	return nil
}

func (etr *EnvironmentTranslationReader) ReadFromScanner(p []byte) (n int, err error) {
	if !etr.isScanReady {
		return 0, errors.New("must be initialized")
	}
	// if etr.buffOut.Len() >= len(p) {
	// 	return etr.buffOut.Read(p)
	// }

	for etr.buffOut.Len() < len(p) && etr.tokenScanner.Scan() {
		if translated, done, missing := ExpandVarString(etr.tokenScanner.Text(), etr.Env); !done {
			return 0, fmt.Errorf("variables were referenced, but not defined: \n%v", missing)
		} else {
			if len(translated) > 0 {
				if _, err := etr.buffOut.WriteString(translated); err != nil {
					return 0, err
				}
			}
		}

		// translated, remaining, readErr := etr.processString(etr.tokenScanner.Text())
		// if len(translated) > 0 {
		// 	if _, err := etr.buffOut.WriteString(translated); err != nil {
		// 		return 0, err
		// 	}
		// }
		// if readErr != nil {
		// 	return 0, readErr
		// }
		// if len(remaining) > 0 {
		// 	//TODO: If we swap to by word or any other token then the remaining may be more relevant? When reading line-by-line it should always be something goofy?
		// 	if _, err := etr.buffOut.WriteString(remaining); err != nil {
		// 		return 0, err
		// 	}
		// 	// etr.buff.WriteString(remaining)
		// }
	}
	if err := etr.tokenScanner.Err(); err != nil {
		//TODO:? err will be nil if nothing went wrong...
		return 0, err
	}

	return etr.buffOut.Read(p)
}

func (etr *EnvironmentTranslationReader) Read(p []byte) (n int, err error) {
	return etr.ReadBuffered(p)
}

func (etr *EnvironmentTranslationReader) ReadBuffered(p []byte) (n int, err error) {

	// Just give them buffered up data if we have enough
	if etr.buffOut.Len() >= len(p) {
		return etr.buffOut.Read(p)
	}

	var nRead, nWrite int
	var errRead, errWrite error

	// Read in the next section
	nRead, errRead = etr.R.Read(p)
	if nRead > 0 {
		nWrite, errWrite = etr.buff.Write(p[0:nRead])

		if nWrite > 0 {
			if processErr := etr.processReadBuffer(); processErr != nil {
				return 0, processErr
			}
		}

		if errWrite != nil {
			return 0, errWrite
		}
	}
	if errRead != nil {
		return n, errRead
	}

	// Now we can read into their original slice from our output buffer
	return etr.buffOut.Read(p)

	// n, err = etr.buff.Write(p);
	// if n > 0 {
	// 	copy(p, etr.buff.Next(n))
	// }
	// if err != nill {
	// 	return n, err
	// }

	// // countNew = copy(etr.buffer[etr.bufferOffset:], p[0:])
	// if n, err = etr.R.Read(p); err != nil {
	// 	return n, err
	// }
	// etr.buffer = append(etr.buffer, p[0:n]...)
	// etr.bufferOffset += n

	//TODO: Try to process the built up string. Do any valid substitutions. Look for what *could* be the start of a variable name near the end.
	//TODO: Track fully processed data so it can be shoved into their buffer slice as room allows
	//TODO: Shove in what processed data we can. Keep the rest in our buffer
	//TODO: Update our index? May require copying things around? How to do efficiently

	return n, nil
}

func (etr *EnvironmentTranslationReader) ReadOriginal(p []byte) (n int, err error) {

	if etr.buffer == nil {
		etr.buffer = make([]byte, len(p), len(p)*3)
		etr.bufferOffset = 0
	}

	// countNew = copy(etr.buffer[etr.bufferOffset:], p[0:])
	if n, err = etr.R.Read(p); err != nil {
		return n, err
	}
	etr.buffer = append(etr.buffer, p[0:n]...)
	etr.bufferOffset += n

	//TODO: Try to process the built up string. Do any valid substitutions. Look for what *could* be the start of a variable name near the end.
	//TODO: Track fully processed data so it can be shoved into their buffer slice as room allows
	//TODO: Shove in what processed data we can. Keep the rest in our buffer
	//TODO: Update our index? May require copying things around? How to do efficiently

	return n, nil
}

// func (etr *EnvironmentTranslationReader) Read(p []byte) (n int, err error) {

// 	if etr.buffer == nil {
// 		etr.buffer = make([]byte, len(p), len(p)*3)
// 		etr.bufferOffset = 0
// 	} else if cap(etr.buffer) < etr.bufferOffset+len(p) {
// 		// Resize the slice to make sure we have room
// 		tempBuffer := make([]byte, etr.bufferOffset+len(p), (etr.bufferOffset+len(p))*2)
// 		countExisting := copy(tempBuffer, etr.buffer[0:etr.bufferOffset])
// 		if countExisting != etr.bufferOffset {
// 			return 0, fmt.Errorf(fmt.Sprintf("incorrect copy count of %d, expected %d", countExisting, etr.bufferOffset))
// 		}
// 	}

// 	// countNew = copy(etr.buffer[etr.bufferOffset:], p[0:])
// 	if n, err = etr.R.Read(etr.buffer[etr.bufferOffset:len(p)]); err != nil {
// 		return n, err
// 	}
// 	etr.bufferOffset += n

// 	//TODO: Try to process the built up string. Do any valid substitutions. Look for what *could* be the start of a variable name near the end.
// 	//TODO: Track fully processed data so it can be shoved into their buffer slice as room allows
// 	//TODO: Shove in what processed data we can. Keep the rest in our buffer
// 	//TODO: Update our index? May require copying things around? How to do efficiently

// 	return n, nil
// }

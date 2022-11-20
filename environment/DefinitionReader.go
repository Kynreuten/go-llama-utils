package environment

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type DefinitionReader struct {
	env    Variables
	reader io.Reader

	tokenScanner bufio.Scanner
	isScanReady  bool

	buffOut bytes.Buffer
}

func NewDefinitionReader(r io.Reader) (translator *DefinitionReader) {
	translator = &DefinitionReader{}
	translator.env = make(Variables, 0, 10)
	translator.reader = bufio.NewReader(r)
	translator.tokenScanner = *bufio.NewScanner(translator.reader)

	translator.isScanReady = true
	return translator
}

const (
	// RegEx pattern to match an expected line of name/value for a variable
	VAR_LINE_REGEX string = `^[ \t]*(?:export)?[ \t]*([A-Za-z][\w-]*)=\"?((?:\\\")*(?:\\\$)*(?:[^\r\n\$\"]*)*(?:(?:\$\{[A-Za-z][\w-]*\})?|(?:\$[A-Za-z][\w-]*)?)*)+\"?$`

	// RegEx pattern to match a full line comment
	COMMENTED_LINE_REGEX       = `^[ \t]*#+(?:[^\n]*)$`
	REGEX_TABSNSPACES          = "[ \t]*"
	REGEX_VARNAME_VALID        = `[A-Za-z][\w-]*`
	REGEX_VARVALUE_VALID       = `(?:\\\")*(?:\\\$)*(?:[^\r\n\$\"]*)*(?:(?:\$\{[A-Za-z][\w-]*\})?|(?:\$[A-Za-z][\w-]*)?)*`
	REGEX_ESC_DBLQUOTE         = `(?:\\\")*`
	REGEX_ESC_SGLQUOTE         = `(?:\\\')*`
	REGEX_ESC_DOLLAR           = `(?:\\\$)*`
	REGEX_VARNAME_FULL_WRAPPED = `(?:\$\{[A-Za-z][\w-]*\})?`
	REGEX_VARNAME_FULL         = `(?:\$[A-Za-z][\w-]*)?`
)

func MakeVarNamePattern() string {
	return fmt.Sprintf("(?:%s)|(?:%s)", REGEX_VARNAME_FULL, REGEX_VARNAME_FULL_WRAPPED)
}
func MakeAllowEscapedPattern() string {
	return fmt.Sprint(REGEX_ESC_DBLQUOTE, REGEX_ESC_SGLQUOTE, REGEX_ESC_DOLLAR)
}
func MakeVariablePattern() string {
	return fmt.Sprintf("%s(%s)?", MakeAllowEscapedPattern(), MakeVarNamePattern())
}
func MakeWrappedVariablePattern(wrapStrings ...string) string {
	vp := MakeVariablePattern()
	sb := strings.Builder{}
	for i, w := range wrapStrings {
		sb.WriteString(fmt.Sprintf("(?:%s%s%s)", w, vp, w))
		if i < len(wrapStrings)-1 {
			sb.WriteRune('|')
		}
	}
	return sb.String()
	// return fmt.Sprintf("(?:%s)|(?:%s)|(?:%s)|(?:%s)", vp, fmt.Sprintf("\"%s\"", vp), fmt.Sprintf("`%s'", vp), fmt.Sprintf("`%s`", vp))
}
func MakeWrappedVariablePatternDefault() string {
	return MakeWrappedVariablePattern("\"", "'", "`")
}

//TODO: Need a way to be able to match a pattern for a variable that's wrapped with any valid quotes or such

func MakeVarLinePattern(validPrefixes []string, validNameValSeparators []string, validValueWrappers []string) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("^%s", REGEX_TABSNSPACES))
	for _, s := range validPrefixes {
		sb.WriteString(s)
	}
	sb.WriteString(REGEX_TABSNSPACES)
	sb.WriteString(fmt.Sprintf("(%s){1}", REGEX_VARNAME_VALID))

	sb.WriteString("(?:")
	for _, s := range validNameValSeparators {
		sb.WriteString(s)
	}
	sb.WriteString("){1}")

	valPattern := MakeWrappedVariablePattern(validValueWrappers...)
	sb.WriteString(fmt.Sprintf("(%s)?", valPattern))

	sb.WriteRune('\n')

	return sb.String()
}

func (etr *DefinitionReader) readFromScanner(p []byte) (n int, err error) {
	if !etr.isScanReady {
		return 0, errors.New("must be initialized")
	}
	varSections := regexp.MustCompile(VAR_LINE_REGEX)
	commentedSection := regexp.MustCompile(COMMENTED_LINE_REGEX)
	var hasMore bool = true
	for hasMore && etr.buffOut.Len() < len(p) {
		// Scan for the next line of data
		hasMore = etr.tokenScanner.Scan()

		//NOTE: May have read some bytes. We're not going to use them if there's a non-EOF error however?
		if !hasMore && etr.tokenScanner.Err() != nil {
			return 0, etr.tokenScanner.Err()
		}

		//TODO: Handle inline comments!
		// Verify it matches our pattern. Ignore commented lines
		sections := varSections.FindStringSubmatch(etr.tokenScanner.Text())
		if len(sections) == 2 {
			// Track the actual name/value of the variable
			etr.env = append(etr.env, Variable{sections[0], sections[1]})
			// Write it out to the normal output buffer
			etr.buffOut.Write(etr.tokenScanner.Bytes())
		} else if len(sections) == 0 && !commentedSection.Match(etr.tokenScanner.Bytes()) {
			return 0, fmt.Errorf("invalid line")
		}
	}

	// Have them read from our output buffer
	return etr.buffOut.Read(p)
}

func (etr *DefinitionReader) Read(p []byte) (n int, err error) {
	return etr.readFromScanner(p)
}

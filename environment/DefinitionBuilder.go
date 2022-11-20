package environment

import (
	"fmt"
	"strings"
)

// Builder for Environment Variable Definitions (or similar).
// Supports writing out lines of name/value pairs such as a .env file's format.
// Fairly flexible, so can be use for a variety of similar formats
type DefinitionBuilder struct {
	// Prefix string to put in front of each name=value entry.
	// Traditionally this would be "export"
	Prefix string
	// string to put between the Prefix and the Name for each line.
	// Only used if Prefix is non-empty
	// Defaults to a single space " "
	PrefixToNameFiller string

	// String to put between the name and value on each line. Normally '='
	NameToValueConnector string

	// Operation to run against each environment variable name when writing it.
	// Should return the final name to use.
	NameHandler func(name string, val string) string
	// Operation to run against each environment variable value when writing it.
	// Should return the final value to use.
	// Often used to wrap with quotes or similar.
	// See the provided ValueHandlerDoubleQuoted and ValueHandlerSingleQuoted functions
	ValueHandler func(name string, val string) string
}

// Creates a default DefinitionBuilder instance to create a normal .env file format
// Names and Values will be written as-is
func NewDefinitionBuilder() *DefinitionBuilder {
	return &DefinitionBuilder{
		Prefix:               "export",
		PrefixToNameFiller:   " ",
		NameToValueConnector: "=",
		NameHandler:          NameHandlerAsIs,
		ValueHandler:         ValueHandlerAsIs,
	}
}

// Simple Key/Value element for tracking Environment Variables
type EnvVariable struct {
	Name  string
	Value string
}

// Puts together a single string representing all of the entries in input
// Returns the full string with a single newline between each entry
func (opts *DefinitionBuilder) BuildString(input []EnvVariable) (str string) {
	sb := strings.Builder{}
	for _, kv := range input {
		if len(opts.Prefix) > 0 {
			sb.WriteString(opts.Prefix)

			if len(opts.PrefixToNameFiller) > 0 {
				sb.WriteString(opts.PrefixToNameFiller)
			}
		}
		sb.WriteString(opts.NameHandler(kv.Name, kv.Value))
		sb.WriteString(opts.NameToValueConnector)
		sb.WriteString(opts.ValueHandler(kv.Name, kv.Value))
		sb.WriteRune('\n')
	}
	return sb.String()
}

// Simple function that returns string n that was given to it.
func StringNoOp(n string) string { return n }

func NameHandlerAsIs(name string, val string) string {
	return name
}
func ValueHandlerAsIs(name string, val string) string {
	return val
}

func NameHandlerDoubleQuoted(name string, val string) string {
	return WrapString(val, "\"")
}
func ValueHandlerDoubleQuoted(name string, val string) string {
	return WrapString(val, "\"")
}
func NameHandlerSingleQuoted(name string, val string) string {
	return WrapString(val, "'")
}
func ValueHandlerSingleQuoted(name string, val string) string {
	return WrapString(val, "'")
}

func WrapString(str string, wrapper string) string {
	return fmt.Sprintf("%s%s%s", wrapper, str, wrapper)
}

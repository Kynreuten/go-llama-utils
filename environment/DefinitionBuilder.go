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

	// Operation to run against each environment variable when writing its Name.
	// Should return the final name to use.
	NameHandler func(enVar Variable) string
	// Operation to run against each environment variable when writing its Value.
	// Should return the final value to use.
	// Often used to wrap with quotes or similar.
	// See the provided ValueHandlerDoubleQuoted and ValueHandlerSingleQuoted functions
	ValueHandler func(enVar Variable) string
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

// Puts together a single string representing all of the entries in input
// Returns the full string with a single newline between each entry
func (opts *DefinitionBuilder) BuildString(input Variables) (str string) {
	sb := strings.Builder{}
	for _, kv := range input {
		if len(opts.Prefix) > 0 {
			sb.WriteString(opts.Prefix)

			if len(opts.PrefixToNameFiller) > 0 {
				sb.WriteString(opts.PrefixToNameFiller)
			}
		}
		sb.WriteString(opts.NameHandler(kv))
		sb.WriteString(opts.NameToValueConnector)
		sb.WriteString(opts.ValueHandler(kv))
		sb.WriteRune('\n')
	}
	return sb.String()
}

// Simple function that returns string n that was given to it.
func StringNoOp(n string) string { return n }

func NameHandlerAsIs(enVar Variable) string {
	return enVar.Name
}
func ValueHandlerAsIs(enVar Variable) string {
	return enVar.Value
}

func NameHandlerDoubleQuoted(enVar Variable) string {
	return WrapString(enVar.Name, "\"")
}
func ValueHandlerDoubleQuoted(enVar Variable) string {
	return WrapString(enVar.Value, "\"")
}
func NameHandlerSingleQuoted(enVar Variable) string {
	return WrapString(enVar.Name, "'")
}
func ValueHandlerSingleQuoted(enVar Variable) string {
	return WrapString(enVar.Value, "'")
}

func WrapString(str string, wrapper string) string {
	return fmt.Sprintf("%s%s%s", wrapper, str, wrapper)
}

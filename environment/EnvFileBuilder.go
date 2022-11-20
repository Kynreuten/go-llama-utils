package environment

import "strings"

type EnvFileBuilderOptions struct {
	// Prefix string to put in front of each name=value entry.
	// This does *NOT* include a trailing space, which may be needed!
	// Traditionally this would be "export "
	Prefix string
	// string to put between the Prefix and the Name for each line. Defaults to a single space " "
	PrefixToNameFiller string

	// Operation to run against each environment variable name.
	// Should return the final name to use.
	NameHandler func(name string, val string) string
	// Operation to run against each environment variable value. Allows wrapping with quotes or similar
	// Should return the final value to use.
	ValueHandler func(name string, val string) string
}

// Creates a default EnvFileBuilderOptions instance with an empty prefix and NoOps for the Name and Value handlers
func NewEnvFileBuilderOptions() *EnvFileBuilderOptions {
	return &EnvFileBuilderOptions{
		Prefix:             "export",
		PrefixToNameFiller: " ",
		NameHandler:        defaultNameHandler,
		ValueHandler:       defaultValueHandler,
	}
}

func defaultNameHandler(name string, val string) string {
	return name
}
func defaultValueHandler(name string, val string) string {
	return val
}

// Simple Key/Value element for tracking Environment Variables
type EnvVariable struct {
	Name  string
	Value string
}

// Simple function that returns string n that was given to it.
func StringNoOp(n string) string { return n }

// Puts together a single string representing all of the entries in input
// Uses the given opts to determine the details of how to write out the data
// Returns the full string with a single newline between each entry
func BuildEnvFileString(input []EnvVariable, opts EnvFileBuilderOptions) (str string) {
	sb := strings.Builder{}
	for _, kv := range input {
		if len(opts.Prefix) > 0 {
			sb.WriteString(opts.Prefix)

			if len(opts.PrefixToNameFiller) > 0 {
				sb.WriteString(opts.PrefixToNameFiller)
			}
		}
		sb.WriteString(opts.NameHandler(kv.Name, kv.Value))
		sb.WriteRune('=')
		sb.WriteString(opts.ValueHandler(kv.Name, kv.Value))
		sb.WriteRune('\n')
	}
	return sb.String()
}

package environment

import "strings"

type EnvFileBuilderOptions struct {
	Prefix       string
	NameHandler  func(n string) string
	ValueHandler func(v string) string
}

// Creates a default EnvFileBuilderOptions instance with an empty prefix and NoOps for the Name and Value handlers
func NewEnvFileBuilderOptions() *EnvFileBuilderOptions {
	return &EnvFileBuilderOptions{
		Prefix:       "",
		NameHandler:  StringNoOp,
		ValueHandler: StringNoOp,
	}
}

// Simple Key/Value element for tracking Environment Variables
type EnvVariable struct {
	Name  string
	Value string
}

// Simple function that returns string n that was given to it.
func StringNoOp(n string) string { return n }

func BuildEnvFileString(input []EnvVariable, opts EnvFileBuilderOptions) (str string) {
	sb := strings.Builder{}
	for _, kv := range input {
		if len(opts.Prefix) > 0 {
			sb.WriteString(opts.Prefix)
		}
		sb.WriteString(opts.NameHandler(kv.Name))
		sb.WriteRune('=')
		sb.WriteString(opts.ValueHandler(kv.Value))
		sb.WriteRune('\n')
	}
	return sb.String()
}

package environment

// Simple Key/Value element for tracking Environment Variables
type Variable struct {
	Name  string
	Value string
}

// Array of type Variable
type Variables []Variable

// Converts this Variables to a VariableMap instance and provides a pointer to it
func (v *Variables) ToMap() *VariableMap {
	m := make(map[string]string, len(*v))
	for _, x := range *v {
		m[x.Name] = x.Value
	}
	xMap := VariableMap(m)
	return &xMap
}

// Map of variable names to values.
type VariableMap map[string]string

// Converts this VariableMap to a Variables instance and provides a pointer to it
func (m *VariableMap) ToVariables() *Variables {
	xMap := map[string]string(*m)
	vars := make([]Variable, len(xMap))
	i := 0
	for n, v := range xMap {
		vars[i] = Variable{n, v}
		i++
	}

	xVars := Variables(vars)
	return &xVars
}

package environment

import (
	"strings"
	"testing"
)

func checkNamesAndValues(t *testing.T, expectedKeys []EnvVariable, envString string) {
	inputReader := strings.NewReader(envString)
	if entries, err := ReadVariables(inputReader); err != nil {
		t.Fail()
	} else {
		if len(expectedKeys) != len(entries) {
			t.Fatalf("incorrect number of entries. want %d, got %d", len(expectedKeys), len(entries))
		}
		for i, s := range expectedKeys {
			wantName := s.Name
			wantValue := s.Value
			gotName := entries[i].Name
			gotValue := entries[i].Value
			if wantName != entries[i].Name {
				t.Fatalf("[%d] want name  `%s`, got `%s` with value '%s'", i, wantName, gotName, gotValue)
			}
			if wantValue != entries[i].Value {
				t.Fatalf("[%d] want value `%s`, got `%s` with name '%s'", i, wantValue, gotValue, gotName)
			}
		}
	}
}

func TestReadEnvironmentMultipleNoQuotes(t *testing.T) {
	expectedKeys := []EnvVariable{
		{"VAR01", "standard"},
		{"VAR02", "slight-variation_with+stuff^\\&@#%()_in~it"},
		{"VAR03", "/noexport/absolute-path"},
		{"VAR04", "../relative-path"},
		{"VAR05", "./local-path"},
		{"VAR06", "./globs/**/path/*.env"},
		{"VAR07", "escaped\\$variablestart"},
	}

	// Default assembly
	opts := NewEnvFileBuilderOptions()

	checkNamesAndValues(t, expectedKeys, BuildEnvFileString(expectedKeys, *opts))
}

func TestReadEnvironmentSingleNoVars(t *testing.T) {
	inputReader := strings.NewReader("exportSOMEVAR+notmuch")
	entries, err := ReadVariables(inputReader)
	if len(entries) > 0 {
		t.Fatalf("got entries:\n%v", entries)
	} else if err != nil {
		t.Fatalf("got an unexpected error:\n%s", err)
	}
}

func TestReadEnvironmentSingle(t *testing.T) {
	inputReader := strings.NewReader("export SOMEVAR=notmuch")
	if entries, err := ReadVariables(inputReader); err != nil {
		t.Fail()
	} else {
		expectedName := "SOMEVAR"
		expectedValue := "notmuch"
		if expectedName != entries[0].Name {
			t.Fatalf("incorrect name %s", entries[0].Name)
		}
		if expectedValue != entries[0].Value {
			t.Fatalf("incorrect value %s", entries[0].Value)
		}
	}
}

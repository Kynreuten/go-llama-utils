package environment

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Reads in the file at the given path for all Environment variable declarations within.
// Each variable found is added to or updated with the latest version in envProcessed.
// Values that contain a known Environment variable will be expanded to contain the variable's value.
// If doPrint == true then detailed debugging information will be printed through the process of reading envData.
func ProcessEnvironmentFile(path string, envProcessed *map[string]string, doPrint bool) {
	file, err := os.Open(path)
	check(err)
	defer file.Close()

	check(ProcessEnvironment(file, envProcessed, doPrint))
}

// Reads in the file at the given path for all Environment variable declarations within.
// Each variable found is added to or updated with the latest version in envProcessed.
// Values that contain a known Environment variable will be expanded to contain the variable's value.
// If doPrint == true then detailed debugging information will be printed through the process of reading envData.
func ProcessEnvironment(r io.Reader, envProcessed *map[string]string, doPrint bool) (err error) {
	rVars := regexp.MustCompile(`\$\{?([\w-]+)\}?`)

	if fileEntries, err := ReadVariables(r); err != nil {
		return err
	} else {
		for _, entry := range fileEntries {
			if doPrint {
				fmt.Printf("## '%s'\n", entry.Name)
			}

			varsFound := rVars.FindAllString(entry.Value, -1)
			if len(varsFound) > 0 {
				if doPrint {
					fmt.Printf("Found %d variables\n", len(varsFound))
					fmt.Println(varsFound)
					fmt.Printf("Expanding: `%s`\n", entry.Value)
				}
				// Attempt to "expand" the found variables
				if expandAttempt, done, neededKeys := ExpandVarString(entry.Value, envProcessed); done {
					// If all were successfully expanded then put the fully expanded value into the envProcessed map under the "name" given.
					//TODO: Duplicate. May need a function?
					if v, ok := (*envProcessed)[entry.Name]; ok {
						// Already exists. Overwrite, but log that fact
						fmt.Printf("-=\t '%s'\n", v)
					}
					// Fully expanded. We can track it as it's "done"
					if doPrint {
						fmt.Printf("+=\t '%s'\n", expandAttempt)
					}
					(*envProcessed)[entry.Name] = expandAttempt
				} else {
					return fmt.Errorf("found %d environment variables referenced that aren't known:\nmissing:\n%v", len(neededKeys), neededKeys)
				}
			} else {
				//TODO: Duplicate! Function?
				if v, ok := (*envProcessed)[entry.Name]; ok {
					// Already exists. Overwrite, but log that fact
					if doPrint {
						fmt.Printf("-=\t '%s'\n", v)
					}
				}
				// There's nothing to expand. Just track it.
				if doPrint {
					fmt.Printf("+=\t '%s'\n", entry.Value)
				}
				(*envProcessed)[entry.Name] = entry.Value
			}
		}
	}
	return nil
}

// ExpandVarString replaces sections of varString that are formatted like environment variables with any matching entries in the given lookup.
// Lookup's keys are expected to be the variable's name, the matching value is what the variable will be replaced with.
// Returns the updated string, whether any replacements were made and an array of variable names that were in varString but don't have values provided in lookup.
func ExpandVarString(varString string, lookup *map[string]string) (replaced string, allTranslated bool, neededKeys []string) {
	// rVars := regexp.MustCompile(`\$\{([\w-]+)\}|\$([\w-]+)`)
	rVars := regexp.MustCompile(`\$\{([A-Za-z]{1}[A-Za-z0-9_-]+)\}|\$([A-Za-z]{1}[A-Za-z0-9_-]+)`)
	neededKeys = []string{}

	missCount := 0
	replaced = rVars.ReplaceAllStringFunc(varString, func(s string) string {
		subs := rVars.FindStringSubmatch(s)
		// k := len(subs[1]) > 0 ? subs[1] : subs[2]
		k := subs[1]
		if len(subs[1]) == 0 {
			k = subs[2]
		}
		if xp, ok := (*lookup)[k]; ok {
			return CleanVarValue(xp)
		} else {
			missCount++
			neededKeys = append(neededKeys, k)
			return CleanVarValue(s)
		}
	})

	return replaced, missCount == 0, neededKeys
}

func FindEndingPartialIndex(target string) (indexOfPartial int, err error) {

	trailFindExpr := regexp.MustCompile(`\$\{?([A-Za-z]{1}[A-Za-z0-9_-]*)?$`)

	if idxes := trailFindExpr.FindStringSubmatchIndex(target); len(idxes) > 0 {
		if len(idxes) > 1 {
			return -1, errors.New("more than one match not possible. regex broken?")
		}

		return idxes[0], nil
	}

	return -1, nil
}

// Cleans up a string that looks like an environment variable.
// Mostly just removes any double-quotes (") from both sides.
func CleanVarValue(varValue string) string {
	return strings.TrimRight(strings.TrimLeft(varValue, `"`), `"`)
}

// NOTE: Named groups for expanding strings...
// const ENV_LINE_REGEX string = `^[ \t]*(?:export)?[ \t]?(?P<key>[A-Z]+[A-Z0-9-_]+)=(?P<value>(?:\"?(?:(?:[\.\w\-:\/\\]*(?:\${[\w-]*\})*)*)\"?)|(?:(?:[\.\w\-:\/\\]*(?:\${[\w-]*\})*)*))$`
const ENV_LINE_REGEX string = `^[ \t]*(?:export)?[ \t]*([A-Za-z][\w-]*)=\"?((?:\\\")*(?:\\\$)*(?:[^\r\n\$\"]*)*(?:(?:\$\{[A-Za-z][\w-]*\})?|(?:\$[A-Za-z][\w-]*)?)*)+\"?$`

func ReadVariables(rIn io.Reader) (envVars []EnvVariable, err error) {
	envVars = make([]EnvVariable, 0, 10)

	scanner := bufio.NewScanner(rIn)
	r := regexp.MustCompile(ENV_LINE_REGEX)
	for scanner.Scan() {
		matches := r.FindStringSubmatch(scanner.Text())
		if len(matches) == 3 {
			envVars = append(envVars, EnvVariable{Name: matches[1], Value: matches[2]})
		} else if len(matches) > 1 {
			return envVars, fmt.Errorf("unexpected matches %d", len(matches))
		}
	}

	if err := scanner.Err(); err != nil {
		return envVars, err
	}

	return envVars, nil
}

// Attempts to read in environment variable key/value pairs from envData.
// path is the path to a file to read enviroment variables from. May be relative or absolute.
// Returns an array of EnvVariable instances. Each representing a Key/value pair of a variable and its value that were found in the file at path.
// If doPrint == true then detailed debugging information will be printed through the process of reading the file.
func ReadVariablesFromFile(path string, doPrint bool) (envVars []EnvVariable) {
	if doPrint {
		fmt.Println("Reading file: ", path)
	}
	file, err := os.Open(path)
	check(err)
	defer file.Close()

	envVars, err = ReadVariables(file)
	//TODO: Shoud probably hand this out as a return value?
	check(err)
	return envVars
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

/*
go mod edit -replace github.com/Kynreuten/go-llama-utils/environment=../environment
go mod tidy
go get github.com/Kynreuten/go-llama-utils
*/

package environment

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

func ProcessEnvironmentFile(p string, envProcessed *map[string]string, doPrint bool) {
	rVars := regexp.MustCompile(`\$\{?([\w-]+)\}?`)

	fileEntries := ReadVariablesFromFile(p, doPrint)

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
				fmt.Println("####--------FAILED TO PROCESS FILE--------####")
				fmt.Printf("Found %d environment variables referenced that aren't known:\n", len(neededKeys))
				fmt.Println("Missing:")
				fmt.Println(neededKeys)
				panic("Unknown Environment variable(s)")
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

/**
* Return:
* string:	{varString} after expansion has been applied
* bool:		Was the string fully expanded? Only true if all variables were known in {lookup}
* []string:	Variables that weren't known.
 */
func ExpandVarString(varString string, lookup *map[string]string) (string, bool, []string) {
	// rVars := regexp.MustCompile(`\$\{([\w-]+)\}|\$([\w-]+)`)
	rVars := regexp.MustCompile(`\$\{([A-Za-z]{1}[A-Za-z0-9_-]+)\}|\$([A-Za-z]{1}[A-Za-z0-9_-]+)`)
	neededKeys := []string{}

	missCount := 0
	replaced := rVars.ReplaceAllStringFunc(varString, func(s string) string {
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

func CleanVarValue(varValue string) string {
	return strings.TrimRight(strings.TrimLeft(varValue, `"`), `"`)
}

// NOTE: Named groups for expanding strings...
// const ENV_LINE_REGEX string = `^[ \t]*(?:export)?[ \t]?(?P<key>[A-Z]+[A-Z0-9-_]+)=(?P<value>(?:\"?(?:(?:[\.\w\-:\/\\]*(?:\${[\w-]*\})*)*)\"?)|(?:(?:[\.\w\-:\/\\]*(?:\${[\w-]*\})*)*))$`
const ENV_LINE_REGEX string = `^[ \t]*(?:export)?[ \t]*([A-Z][\w-]*)=((?:\"?(?:[^\r\n\$\"]*(?:\$\{?[A-Z][\w-]*\}?)*)+\"?)|(?:[\.\w\-:\/\\]*(?:\${[A-Z][\w-]*\})*)+){1}$`

func ReadVariablesFromFile(path string, doPrint bool) []EnvVariable {
	envVars := []EnvVariable{}

	if doPrint {
		fmt.Println("Reading file: ", path)
	}
	file, err := os.Open(path)
	check(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	r := regexp.MustCompile(ENV_LINE_REGEX)
	for scanner.Scan() {
		matches := r.FindStringSubmatch(scanner.Text())
		// fmt.Println("Read line: \t", scanner.Text())
		if len(matches) == 3 {
			if doPrint {
				fmt.Println("Variable=\t", matches[1])
				fmt.Println("Value=\t\t", matches[2])
			}
			envVars = append(envVars, EnvVariable{Name: matches[1], Value: matches[2]})
		} else if len(matches) > 1 {
			fmt.Println("UNEXPECTED MATCHES? ", len(matches))
		}
	}
	if doPrint {
		fmt.Println("#-###----------------###-#")
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return envVars
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type EnvVariable struct {
	Name  string
	Value string
}

/*
go mod edit -replace github.com/Kynreuten/go-llama-utils/environment=../environment
go mod tidy
go get github.com/Kynreuten/go-llama-utils
*/

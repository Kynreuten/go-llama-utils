package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Kynreuten/go-llama-utils/environment"
)

var _opts OperationOptions

func init() {
	_opts = CreateDefaultOperationOptions()
	//TODO: Consider subcommands using NewFlagSet if we want multiple approaches?
	//https://gobyexample.com/command-line-subcommands
	// Look at flagged options
	flag.StringVar(&_opts.CommandPath, "cmd", "", "Command to execute. Must be a valid path. Can be relative or absolute.")
	flag.Var(&_opts.CommandArgsRaw, "cmdArg", "Arguments for the command itself. You may supply multiple of these. Should include flag and value together with an equals sign between them. No equals if it has no value. \nEx 'loglevel=debug' or 'something=nope'")
	//TODO: Add other flags for various settings on what we want reported or done
	flag.BoolVar(&_opts.UseStdOut, "useStdOut", true, "True if you want Standard Output to be shown in this terminal")
	flag.BoolVar(&_opts.UseStdErr, "useStdErr", true, "True if you want Standard Error to be shown in this terminal")
	flag.BoolVar(&_opts.DoLogDebug, "debug.main", false, "True if you want most debug info displayed")
	flag.BoolVar(&_opts.DoLogEnv, "debug.env", false, "True if you want to log all Environment data")
	flag.BoolVar(&_opts.IsTest, "test", false, "True if you want to only show what would be done and exit")

	flag.Parse()
	_opts.Globs = flag.Args()
}

func main() {
	// Verify the target command appears valid.
	targetCmd, err := exec.LookPath(_opts.CommandPath)
	if err != nil { // errors.Is(err, os.ErrNotExist) {
		log.Fatalf("Failure looking at command: \"%s\"\n%s", targetCmd, err.Error())
	}
	if err := processEnvGlobs(&_opts); err != nil {
		log.Fatal(err.Error())
	}
	if err := processCommandArgs(&_opts); err != nil {
		log.Fatal(err.Error())
	}

	fmt.Println("Passed Command: ", targetCmd)
	if _opts.DoLogDebug {
		fmt.Println("Command Arguments: ")
		fmt.Println(_opts.CommandArgs)
	}
	if _opts.DoLogDebug {
		fmt.Println("Env Paths: ")
		fmt.Println(_opts.EnvPaths)
	}

	// Start making the actual command to run. We assume that all text before a space is the path to the command. Anything else is space-delimited arguments for it
	cmd := exec.Command(targetCmd, _opts.CommandArgs...)

	// Environment variables that have been completely processed
	envProcessed := make(map[string]string)
	// Attempt to read in the contents of each environment file
	for _, p := range _opts.EnvPaths {
		environment.ProcessEnvironmentFile(p, &envProcessed, _opts.DoLogEnv)

		if _opts.DoLogEnv {
			fmt.Println("####----------------####")
		}
	}

	cmd.Env = make([]string, len(envProcessed))
	fmt.Println("Environment:")
	if len(envProcessed) > 0 {
		i := 0
		for k, v := range envProcessed {
			cmd.Env[i] = fmt.Sprintf("%s=%s", k, v)
			fmt.Printf("`%s`\n", cmd.Env[i])
			i++
		}
	} else {
		fmt.Println("No Environment variables found")
	}

	//TODO: Support alternative places to put it? Maybe to a file?
	if _opts.UseStdOut {
		cmd.Stdout = os.Stdout
	}
	//TODO: Support alternative places to put it? Maybe to a file?
	if _opts.UseStdErr {
		cmd.Stderr = os.Stderr
	}

	fmt.Println("####--------++--------####")
	if _opts.IsTest {
		fmt.Println("Command to run:")
		fmt.Println(cmd.String())
		fmt.Printf("Args: %+q\n", strings.Join(cmd.Args, ","))
		fmt.Printf("Env:\n%+q\n", strings.Join(cmd.Environ(), ","))
	} else {
		if _opts.DoLogDebug {
			fmt.Printf("Running Command:\n%s\n", cmd.String())
		}
		cmd.Run()
	}
}

// func check(e error) {
// 	if e != nil {
// 		panic(e)
// 	}
// }

func processEnvGlobs(opts *OperationOptions) error {
	allGlobs := opts.Globs

	// Track our other flags
	if opts.DoLogDebug {
		fmt.Println("Globs:")
		fmt.Println(strings.Join(allGlobs, ", "))
	}

	fmt.Println("Command: ", opts.CommandPath)
	//fmt.Println("Environment files Glob: ", *envGlobPtr)
	// fmt.Println("Extra flags: ", flag.Args())

	// Find matching file paths
	allPaths := []string{}
	for _, p := range allGlobs {
		if opts.DoLogDebug {
			fmt.Printf("Checking glob: %s\n", p)
		}
		matches, err := filepath.Glob(p)
		if err != nil {
			fmt.Println(err)
			panic("failed to read target file path globs")
		} else if len(matches) == 0 {
			for _, v := range matches {
				if _, err := os.Stat(v); err != nil {
					if errors.Is(err, os.ErrNotExist) {
						return fmt.Errorf("target file not found:\n%s", v)
					} else {
						return fmt.Errorf("failed to examine file:\n%s\nerror:\n%s", v, err)
					}
				}
			}
			if opts.DoLogDebug {
				fmt.Printf("Found count: %d\n", len(matches))
			}
		}
		allPaths = append(allPaths, matches...)
	}

	// _opts.Globs = allGlobs
	opts.EnvPaths = allPaths
	return nil
}

func processCommandArgs(opts *OperationOptions) error {
	opts.CommandArgs = make([]string, len(opts.CommandArgsRaw))
	rEntry := regexp.MustCompile(`^([A-Za-z]{1}[A-Za-z0-9_\-\.]*)(?:[=:]{1}([^\r\n]+))?$`)

	for i, a := range opts.CommandArgsRaw {
		curr := rEntry.FindStringSubmatch(a)
		if len(curr) == 3 {
			// Single character args we assume should have a single dash. Others get double dashes
			prefix := "--"
			if len(curr[1]) == 1 {
				prefix = "-"
			}
			if len(curr[2]) > 0 {
				opts.CommandArgs[i] = fmt.Sprintf("%s%s %s", prefix, curr[1], curr[2])
			} else {
				opts.CommandArgs[i] = fmt.Sprintf("%s%s", prefix, curr[1])
			}
		} else if len(curr) == 2 {
			opts.CommandArgs[i] = fmt.Sprintf("-%s", curr[1])
		} else {
			return fmt.Errorf("invalid command argument was supplied: \n%s", a)
		}
	}

	return nil
}

// func run() {
// 	// TODO: pass in the command and any args...
// 	cmd := exec.Command("ansible-playbook", args...)
// 	cmd.Env = append(cmd.Env, "MY_VAR=some_value")
// 	cmd.Env = os.Environ()
// }

type CommandArguments []string

func (i *CommandArguments) String() string {
	return strings.Join(*i, ", ")
}
func (i *CommandArguments) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type OperationOptions struct {
	Globs    []string
	EnvPaths []string

	CommandPath    string
	CommandArgsRaw CommandArguments
	CommandArgs    []string

	IsTest     bool
	DoLogDebug bool
	DoLogEnv   bool
	UseStdOut  bool
	UseStdErr  bool
}

func CreateDefaultOperationOptions() OperationOptions {
	opts := OperationOptions{
		CommandPath: "",
		IsTest:      false,
		DoLogDebug:  false,
		DoLogEnv:    false,
		UseStdOut:   true,
		UseStdErr:   true,
	}
	return opts
}

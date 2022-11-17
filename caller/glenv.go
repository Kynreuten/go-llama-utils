// #exec command
// Executes a given process as though from a shell. Utilizes provided Environment Variables to transform the arguments for the target process.
//
// #read command
// Reads in a source string and transforms Environment Variables that are found in it into their values from any provided Environment Files.

package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Kynreuten/go-llama-utils/environment"
)

var _opts OperationOptions

// func initOriginal() {
// 	_opts = CreateDefaultOperationOptions()
// 	//TODO: Consider subcommands using NewFlagSet if we want multiple approaches?
// 	//https://gobyexample.com/command-line-subcommands
// 	// Look at flagged options
// 	flag.StringVar(&_opts.CommandPath, "cmd", "", "Command to execute. Must be a valid path. Can be relative or absolute.")
// 	flag.Var(&_opts.CommandArgsRaw, "cmdArg", "Arguments for the command itself. You may supply multiple of these. Should include flag and value together with an equals sign between them. No equals if it has no value. \nEx 'loglevel=debug' or 'something=nope'")
// 	//TODO: Add other flags for various settings on what we want reported or done
// 	flag.BoolVar(&_opts.UseStdOut, "useStdOut", true, "True if you want Standard Output to be shown in this terminal")
// 	flag.BoolVar(&_opts.UseStdErr, "useStdErr", true, "True if you want Standard Error to be shown in this terminal")
// 	flag.BoolVar(&_opts.DoLogDebug, "debug.main", false, "True if you want most debug info displayed")
// 	flag.BoolVar(&_opts.DoLogEnv, "debug.env", false, "True if you want to log all Environment data")
// 	flag.BoolVar(&_opts.IsTest, "test", false, "True if you want to only show what would be done and exit")

// 	flag.Parse()
// 	_opts.Globs = flag.Args()
// }

const (
	TYPE_EXEC = "exec"
	TYPE_READ = "read"
)

func init() {
	_opts = CreateDefaultOperationOptions()

	execFlags := flag.NewFlagSet(TYPE_EXEC, flag.ExitOnError)
	execFlags.StringVar(&_opts.CommandPath, "cmd", "", "Command to execute. Must be a valid path. Can be relative or absolute.")
	execFlags.Var(&_opts.CommandArgsRaw, "a", "Arguments for the command itself. You may supply multiple of these. Should include flag and value together with an equals sign between them. No equals if it has no value. \nEx 'loglevel=debug' or 'something=nope'")
	addStandardOptions(execFlags)

	readFlags := flag.NewFlagSet(TYPE_READ, flag.ExitOnError)
	readFlags.StringVar(&_opts.TargetInPath, "i", "", "Path to the file to read as our input string. Must be a valid path. Can be relative or absolute. If not provided then standard input is assumed.")
	readFlags.StringVar(&_opts.TargetOutPath, "o", "", "Path to the file to write the converted string to. If not provided then standard input is assumed. Must be a valid path. Can be relative or absolute. Sent to Standard Out if not specified")
	addStandardOptions(readFlags)

	if len(os.Args) < 2 {
		fmt.Println("Expected a subcommand of 'exec', or 'read")
		os.Exit(1)
	}

	_opts.Type = os.Args[1]
	switch _opts.Type {
	case TYPE_EXEC:
		fmt.Println("Exec Subcommand chosen.")
		execFlags.Parse(os.Args[2:])
		_opts.Globs = execFlags.Args()
	case TYPE_READ:
		fmt.Println("Read Subcommand chosen.")
		readFlags.Parse(os.Args[2:])
		_opts.Globs = readFlags.Args()
	default:
		fmt.Printf("Unknown subcommand '%s'. Expecting '%s', or '%s", _opts.Type, TYPE_EXEC, TYPE_READ)
		fmt.Println(os.Args)
		os.Exit(1)
	}
}

func addStandardOptions(targetFlag *flag.FlagSet) {
	targetFlag.BoolVar(&_opts.UseStdOut, "useStdOut", true, "True if you want Standard Output to be shown in this terminal")
	targetFlag.BoolVar(&_opts.UseStdErr, "useStdErr", true, "True if you want Standard Error to be shown in this terminal")
	targetFlag.BoolVar(&_opts.DoLogDebug, "debug.main", false, "True if you want most debug info displayed")
	targetFlag.BoolVar(&_opts.DoLogEnv, "debug.env", false, "True if you want to log all Environment data")
	targetFlag.BoolVar(&_opts.IsTest, "test", false, "True if you want to only show what would be done and exit")
}

func main() {
	switch _opts.Type {
	case TYPE_EXEC:
		executeCmdAction()
	case TYPE_READ:
		transformAction()
	}
}

// Attempts to use any given environment information to transform the given data
func transformAction() {
	if processEnv, err := readEnv(); err != nil {
		log.Fatal(err)
	} else {
		// Open reader to input file
		if fIn, err := os.Open(_opts.TargetInPath); err != nil {
			log.Fatal(err)
		} else {
			defer fIn.Close()
			// Open writer to output file
			if fOut, err := os.Open(_opts.TargetOutPath); err != nil {
				log.Fatal(err)
			} else {
				defer fOut.Close()
				// Prep our translator
				tr := environment.NewTranslator(processEnv, fIn)

				bfOut := bufio.NewWriter(fOut)
				if _, err := bfOut.ReadFrom(tr); err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}

// Attempts to read and process environment variables in the files referenced in _opts.EnvPaths
// Returns a pointer to a map of environment variable keys to values as strings that were read in.
func readEnv() (*map[string]string, error) {

	if err := processEnvGlobs(&_opts); err != nil {
		return nil, err
	}
	if _opts.DoLogDebug {
		fmt.Println("Env Paths: ")
		fmt.Println(_opts.EnvPaths)
	}

	// Environment variables that have been completely processed
	envProcessed := make(map[string]string)
	// Attempt to read in the contents of each environment file
	for _, p := range _opts.EnvPaths {
		environment.ProcessEnvironmentFile(p, &envProcessed, _opts.DoLogEnv)

		if _opts.DoLogEnv {
			fmt.Println("####----------------####")
		}
	}

	envEntries := make([]string, len(envProcessed))
	fmt.Println("Environment:")
	if len(envProcessed) > 0 {
		i := 0
		for k, v := range envProcessed {
			envEntries[i] = fmt.Sprintf("%s=%s", k, v)
			fmt.Printf("`%s`\n", envEntries[i])
			i++
		}
	} else {
		return nil, errors.New("no environment variables found")
	}

	return &envProcessed, nil
}

// Puts together a list of the given envProcessed entries. If doPrint is true then these will be printed out while assembling the array of values
// Returns a pointer to an array of all entries from envProcessed
func listEnv(envProcessed map[string]string, doPrint bool) (*[]string, error) {

	envEntries := make([]string, len(envProcessed))
	if doPrint {
		fmt.Println("Environment:")
	}
	if len(envProcessed) > 0 {
		i := 0
		for k, v := range envProcessed {
			envEntries[i] = fmt.Sprintf("%s=%s", k, v)
			if doPrint {
				fmt.Printf("`%s`\n", envEntries[i])
			}
			i++
		}
		return &envEntries, nil
	} else {
		return nil, errors.New("no environment variables found")
	}
}

// Attempts to execute the command that was given via program arguments
func executeCmdAction() {
	// Verify the target command appears valid.
	targetCmd, err := exec.LookPath(_opts.CommandPath)
	if err != nil { // errors.Is(err, os.ErrNotExist) {
		log.Fatalf("Failure looking at command: \"%s\"\n%s", targetCmd, err.Error())
	}
	if err := processCommandArgs(&_opts); err != nil {
		log.Fatal(err.Error())
	}

	fmt.Println("Passed Command: ", targetCmd)
	if _opts.DoLogDebug {
		fmt.Println("Command Arguments: ")
		fmt.Println(_opts.CommandArgs)
	}

	// Environment variables that have been completely processed
	var envProcessed, readErr = readEnv()
	if readErr != nil {
		log.Fatal(readErr)
	}

	// Start making the actual command to run. We assume that all text before a space is the path to the command. Anything else is space-delimited arguments for it
	cmd := exec.Command(targetCmd, _opts.CommandArgs...)

	listEnv(*envProcessed, _opts.DoLogEnv)

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
	rEntry := regexp.MustCompile(`^(?:([A-Za-z]{1}[A-Za-z0-9_\-\.]*)[=:]?)?([^\r\n]+)?$`)

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
			return fmt.Errorf("invalid command argument was supplied (#%d): \n%s\nExtra?%s", len(curr), a, curr)
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

// Wrapper type for a string array. Allows for an array of arguments with the same name
type CommandArguments []string

// Put together the CommandArguments into a single string
func (i *CommandArguments) String() string {
	return strings.Join(*i, ", ")
}

// Build up the command arguments
func (i *CommandArguments) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// Tracks information on what Operation should be performed
type OperationOptions struct {
	// Type of Operation this is for
	Type string
	// Globs that describe the Environment files to process
	Globs []string
	// Paths to the actual Environment files to process
	EnvPaths []string

	// Path to the command to execute
	CommandPath string
	// Raw arguments for the command that were passed in
	CommandArgsRaw CommandArguments
	// Final processed arguments that will be passed to the command
	CommandArgs []string

	// Path to a file to read in and process for Environment Variables.
	TargetInPath string
	// Reader to read in data to transform
	TargetInReader io.Reader
	// Path where the updated version of TargetInPath should be written.
	TargetOutPath string
	// Writer to write transformed data to
	TargetOutWriter io.Writer

	//TODO: Track the Input and Output streams to use. May allow removing some other args?

	// Is this just a test run?
	// If true then the operation requested won't be performed, but all elements of it will be logged as if it were
	IsTest bool
	// Should detailed logging information be provided everywhere possible?
	DoLogDebug bool
	// Should the processing of Environment variables be logged?
	DoLogEnv bool
	// Should the calling shell's Standard Out be used as Standard Out for the called process?
	UseStdOut bool
	// Should the calling shell's Standard Error be used as Standard Error for the called process?
	UseStdErr bool
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

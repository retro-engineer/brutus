package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type argStringArray []string

func (a *argStringArray) String() string {
	return strings.Join(*a, " ")
}
func (a *argStringArray) Set(value string) error {
	*a = append(*a, value)
	return nil
}

type commandResult struct {
	output string
	values []string
	err    error
}

func createTasks(values []string, files []string, commandTemplate Shell, done <-chan struct{}, commands chan<- Shell, errc chan<- error) error {
	if len(files) == 0 {
		command := commandTemplate
		command.Values = values
		select {
		case commands <- command:
		case <-done:
			err := errors.New("command generation cancelled")
			errc <- err
			return err
		}
		return nil
	}
	file, err := os.Open(files[0])
	if err != nil {
		errc <- err
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		value := scanner.Text()
		newValues := append(values, value)
		err := createTasks(newValues, files[1:], commandTemplate, done, commands, errc)
		if err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		errc <- err
		return err
	}
	return nil
}

func worker(done <-chan struct{}, commands <-chan Shell, results chan<- commandResult) {
	for c := range commands {
		cmd := c.Command()
		// execute command, gather output and errors
		out, err := cmd.CombinedOutput()
		select {
		case results <- commandResult{string(out), c.Values, err}:
		case <-done:
			return
		}
	}
}

func executeCommands(files []string, commandTemplate Shell, numWorkers int, verbose bool, tries bool, positive bool, progress bool, success []string, failure []string) error {
	done := make(chan struct{})
	defer close(done)
	commands := make(chan Shell)
	//defer close(commands)
	errc := make(chan error, 1)
	defer close(errc)

	go func() {
		_ = createTasks(nil, files, commandTemplate, done, commands, errc)
		close(commands)
	}()

	// Start a fixed number of goroutines to execute commands.
	results := make(chan commandResult)
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			worker(done, commands, results)
			wg.Done()
		}()
	}
	go func() {
		// Wait for all the workers to be done
		wg.Wait()
		close(results)
	}()

	progressString := ""
	for r := range results {
		printOutput := true
		if len(success) > 0 || len(failure) > 0 {
			printOutput = false
		}
		foundSuccess := false
		for _, s := range success {
			if strings.Contains(r.output, s) {
				foundSuccess = true
			}
		}
		foundFailure := false
		for _, s := range failure {
			if strings.Contains(r.output, s) {
				foundFailure = true
			}
		}
		if progress {
			tries = false
			fmt.Print("\r")
			for range progressString {
				fmt.Print(" ")
			}
			fmt.Print("\r")
			progressString = fmt.Sprintf("[?] Trying: %s", strings.Join(r.values, "   "))
			fmt.Print(progressString)
		}
		if tries {
			fmt.Printf("[?] Trying: %s\n", strings.Join(r.values, "   "))
		}
		printValues := false
		if foundFailure {
			printValues = false
		} else {
			if foundSuccess || positive {
				printValues = true
			}
		}
		if printValues {
			if progress {
				fmt.Print("\n")
			}
			fmt.Printf("[+] Success: %s\n", strings.Join(r.values, "   "))
		}
		if printOutput || verbose {
			if progress {
				fmt.Print("\n")
			}
			fmt.Print(r.output)
		}
	}
	if progress {
		fmt.Print("\n")
	}
	select {
	case err := <-errc:
		return err
	default:
		return nil
	}
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: %s [OPTION] COMMANDS

Options:
  -file:     array of files to read contents from
  -shell:    execution shell (default: %s)
  -shellarg: execution shell argument array (default: %s)
  -success:  strings that indicate success
  -failure:  strings that indicate failure. These have precedense over success strings.
  -fuzz:     the identifier that is replaced by the file contents.
             The curly braces are replaced by '', '2', '3', etc. (default: FUZ{}Z)
  -threads:  the number of concurrent workers (default: %v)
  -tries:    print the tried combinations
  -positive: treat an absense of failure as a success
  -progress: print progress
  -verbose:  verbose output

Examples:
  SMB   user/pass bruteforce:  %s -file users.txt -file passwords.txt -success IP smbmap -u FUZZ -p FUZ2Z -H 10.10.10.179
  MySQL user/pass bruteforce:  %s -file users.txt -file passwords.txt -failure 'Access denied' mysql -u\'FUZZ\' -p\'FUZ2Z\' --host=127.0.0.1
  SSH   user/pass bruteforce:  %s -file users.txt -file passwords.txt -success Success ssh-test 10.10.10.179 22 FUZZ 'FUZ2Z'
`, os.Args[0], DefaultShell, strings.Join(DefaultArgs, " "), 10, os.Args[0], os.Args[0], os.Args[0])
	}

	// Flags
	var shellCmd string
	flag.StringVar(&shellCmd, "shell", DefaultShell, "execution shell")

	var customShellArgs argStringArray
	flag.Var(&customShellArgs, "shellarg", "execution shell argument array (default: "+strings.Join(DefaultArgs, " ")+")")

	var files argStringArray
	flag.Var(&files, "file", "array of files to read contents from")

	var fuzzTerm string
	flag.StringVar(&fuzzTerm, "fuzz", "FUZ{}Z", "the identifier that is replaced by the file contents. The curly braces are replaced by '', '2', '3', etc.")

	var threadCount int
	flag.IntVar(&threadCount, "threads", 10, "the number of concurrent workers")

	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "verbose output")

	var tries bool
	flag.BoolVar(&tries, "tries", false, "print the tried combinations")

	var positive bool
	flag.BoolVar(&positive, "positive", false, "treat an absense of failure as a success")

	var progress bool
	flag.BoolVar(&progress, "progress", false, "print progress")

	var success argStringArray
	flag.Var(&success, "success", "strings that indicate success")

	var failure argStringArray
	flag.Var(&failure, "failure", "strings that indicate failure. These have precedense over success strings.")

	flag.Parse()

	if len(files) == 0 || len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var shellArgs []string
	if len(customShellArgs) == 0 {
		shellArgs = DefaultArgs
	} else {
		shellArgs = customShellArgs
	}
	commandTemplate := Shell{
		Shell:    shellCmd,
		Args:     shellArgs,
		Cmd:      strings.Join(flag.Args(), " "),
		FuzzTerm: fuzzTerm,
	}

	fmt.Println(`===============================================================
Brutus
===============================================================`)
	fmt.Printf("[+] Command:        %s %s %s\n", commandTemplate.Shell, strings.Join(commandTemplate.Args, " "), commandTemplate.Cmd)
	if verbose {
		fmt.Println("[+] Verbose:        true")
	}
	if tries {
		fmt.Println("[+] Print tried combinations:    true")
	}
	fmt.Printf("[+] Threads:        %s\n", strconv.Itoa(threadCount))
	fmt.Println("[+] Wordlists:")
	for i, f := range files {
		extraSpace := ""
		if i == 0 {
			extraSpace = " "
		}
		fmt.Printf("                    %s: %s%s\n", commandTemplate.Term(i), extraSpace, f)
	}
	if len(success) > 0 {
		fmt.Printf("[+] Success:        %s\n", strings.Join(success, " "))
	}
	if len(failure) > 0 {
		fmt.Printf("[+] Failure:        %s\n", strings.Join(failure, " "))
	}
	/*	[+] Timeout:        10s*/
	fmt.Println(`===============================================================
Starting brutus
===============================================================`)
	err := executeCommands(files, commandTemplate, threadCount, verbose, tries, positive, progress, success, failure)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
}

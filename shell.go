package main

import (
	"os/exec"
	"strconv"
	"strings"
)

// Shell : A struct that contains all the shell parameters and fuzz terms.
type Shell struct {
	Shell    string
	Args     []string
	Cmd      string
	FuzzTerm string
	Values   []string
}

// Command : Parses stored values and returns an executable command.
func (s Shell) Command() exec.Cmd {
	cmd := s.Cmd
	for i, v := range s.Values {
		needle := s.Term(i)
		cmd = strings.ReplaceAll(cmd, needle, v)
	}
	args := append(s.Args, cmd)
	command := exec.Command(s.Shell, args...)
	return *command
}

// Term : Generates a fuzz term given an index. The curly braces are replaced by '', '2', '3', etc.
func (s Shell) Term(index int) string {
	if index == 0 {
		return strings.ReplaceAll(s.FuzzTerm, "{}", "")
	}
	return strings.ReplaceAll(s.FuzzTerm, "{}", strconv.FormatInt(int64(index+1), 10))
}

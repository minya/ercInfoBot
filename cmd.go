package main

import (
	"fmt"
	"regexp"
)

// ParseCommand receives telegram cmd string and produces Command structure
func ParseCommand(cmdStr string) (Command, error) {
	reCommand, _ := regexp.Compile("(/?[\\w\\.;@,!@#$&^-_=*\\+]+)")
	match := reCommand.FindAllStringSubmatch(cmdStr, -1)

	cmd := Command{}
	if len(match) == 0 {
		return cmd, fmt.Errorf("Unknown command: %v", cmdStr)
	}

	cmd.Command = match[0][0]
	switch cmd.Command {
	case "/reg":
		cmd.Args = make([]string, 3, 3)
		cmd.Args[0] = match[1][0]
		cmd.Args[1] = match[2][0]
		cmd.Args[2] = match[3][0]
	case "/receipt":
		cmd.Args = make([]string, 0, 0)
	case "/notify":
		cmd.Args = make([]string, 1, 1)
		cmd.Args[0] = match[0][0]
	case "/help":
		cmd.Args = make([]string, 0, 0)
	case "/get":
		cmd.Args = make([]string, 0, 0)
	default:
		return cmd, fmt.Errorf("Unknown command: %v", cmd.Command)
	}

	return cmd, nil
}

// Command structure:
// Command - name of command (/reg, /help, etc)
// Args - arguments
type Command struct {
	Command string
	Args    []string
}

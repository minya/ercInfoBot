package main

import (
	"fmt"
	"testing"
)

func TestParseReg_IgnoreArgs_IfMore(t *testing.T) {
	command, err := ParseCommand("/reg a_aaa@a.com qwe123QWE!@# 123456")
	assureCorrectRegCmd(command, err, t)
}

func TestParseReg_ExactArgs(t *testing.T) {
	command, err := ParseCommand("/reg a_aaa@a.com qwe123QWE!@# 123456")
	assureCorrectRegCmd(command, err, t)
}

func TestParseNotify(t *testing.T) {
	command, _ := ParseCommand("/notify on")
	if command.Command != "/notify" {
		t.Error("expected /notify, but got ", command.Command)
	}
}

func TestParseInfoCommands(t *testing.T) {
	var doTest = func(t *testing.T, cmd string) {
		command, err := ParseCommand(fmt.Sprintf("%v account1", "/receipt"))
		if err != nil {
			t.Error("Error while parse command")
		}

		if command.Command != "/receipt" {
			t.Error("Wrong command name")
		}

		if command.Args[0] != "account1" {
			t.Error("Wrong argument")
		}
	}
	doTest(t, "/receipt")
	doTest(t, "/get")
}

func assureCorrectRegCmd(command Command, err error, t *testing.T) {
	if err != nil {
		t.Error("Error should not have happened")
	}

	got := command.Command
	if got != "/reg" {
		t.Error("expected /reg, but got ", got)
	}
	if len(command.Args) != 2 {
		t.Error("Expected 2 args")
	}
	if command.Args[0] != "a_aaa@a.com" {
		t.Error("expected a1, but got ", command.Args[0])
	}
	if command.Args[1] != "qwe123QWE!@#" {
		t.Error("expected qwe123QWE!@#, but got ", command.Args[1])
	}
}

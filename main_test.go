package main

import (
	"testing"
)

func TestParseReg(t *testing.T) {
	command, _ := ParseCommand("/reg a_aaa@a.com qwe123QWE!@# 123456")
	got := command.Command
	if got != "/reg" {
		t.Error("expected /reg, but got ", got)
	}
	if command.Args[0] != "a_aaa@a.com" {
		t.Error("expected a1, but got ", command.Args[0])
	}
	if command.Args[1] != "qwe123QWE!@#" {
		t.Error("expected qwe123QWE!@#, but got ", command.Args[1])
	}
	if command.Args[2] != "123456" {
		t.Error("expected 123456, but got ", command.Args[2])
	}
}

func TestParseNotify(t *testing.T) {
	command, _ := ParseCommand("/notify on")
	if command.Command != "/notify" {
		t.Error("expected /notify, but got ", command.Command)
	}

}

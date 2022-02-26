package main

import (
	"fmt"
	"strings"
)

type Command struct {
	Verb     string
	Argument string
}

type Response struct {
	Code    string
	Message string
}

func ParseCommand(what string) (Command, error) {
	command := Command{}
	index := strings.IndexRune(what, ' ')
	if index == -1 {
		command.Verb = strings.ToUpper(what)
		command.Argument = ""
		return command, nil
	}
	command.Verb = strings.ToUpper(what[:index])
	command.Argument = what[index+1:]
	return command, nil
}

func NewResponse(code string, message string) Response {
	return Response{Code: code, Message: message}
}

func (v Response) ToString() string {
	return fmt.Sprintf("%s %s", v.Code, v.Message)
}

package shellError

import (
	"fmt"
	"gosh/config"
)

type shellErrorObj struct {
	command *string
	msg     string
	blame   *string
}

type ShellError = *shellErrorObj

func (e ShellError) Error() string {
	if e.command == nil {
		return fmt.Sprintf("%s: %s", config.AppConfig.ShellName, e.msg)
	}
	if e.blame == nil {
		return fmt.Sprintf("%s: %s: %s", config.AppConfig.ShellName, *e.command, e.msg)
	}
	return fmt.Sprintf("%s: %s: %s: %s", config.AppConfig.ShellName, *e.command, *e.blame, e.msg)
}

func New(msg string) ShellError {
	return &shellErrorObj{command: nil, msg: msg, blame: nil}
}

func NewCommand(command string, msg string) ShellError {
	return &shellErrorObj{command: &command, msg: msg, blame: nil}
}

func NewCommandWithBlame(command string, msg string, blame string) ShellError {
	return &shellErrorObj{command: &command, msg: msg, blame: &blame}
}

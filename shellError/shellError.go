package shellError

import "fmt"

type shellErrorObj struct {
	command string
	msg     string
	blame   *string
}

type ShellError = *shellErrorObj

func (e ShellError) Error() string {
	if e.blame == nil {
		return fmt.Sprintf("%s: %s: %s", "gosh", e.command, e.msg)
	}
	return fmt.Sprintf("%s: %s: %s: %s", "gosh", e.command, *e.blame, e.msg)
}

func New(command string, msg string) ShellError {
	return &shellErrorObj{command: command, msg: msg, blame: nil}
}

func NewWithBlame(command string, msg string, blame string) ShellError {
	return &shellErrorObj{command: command, msg: msg, blame: &blame}
}

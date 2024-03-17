package runner

import (
	"bufio"
	"fmt"
	. "gosh/config"
	. "gosh/expansions"
	. "gosh/pathSearcher"
	. "gosh/tokenizer"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// -- BUILT-INS --

type BuiltinFunc = func(...string) error

func exit(args ...string) error {
	fmt.Println("exit")
	for _, arg := range args {
		code, err := strconv.Atoi(arg)
		if err != nil {
			os.Exit(code)
		}
		os.Exit(0)
	}
	os.Exit(0)
	return nil
}

func cd(args ...string) error {
	var directory string
	if len(args) > 1 {
		return fmt.Errorf("cd: too many arguments")
	} else if len(args) == 1 && args[0] != "~" {
		directory = args[0]
	} else {
		directory = AppConfig.UserHomeDir
	}

	var err error
	if err = syscall.Chdir(directory); err != nil {
		return fmt.Errorf("cd: No such file or directory: %s", directory)
	}
	AppConfig.AbsolutePath, err = os.Getwd()
	tildifyCdPath()
	return nil
}

func pwd(_ ...string) error {
	fmt.Println(AppConfig.AbsolutePath)
	return nil
}

func tildifyCdPath() {
	AppConfig.DisplayCdPath = AppConfig.AbsolutePath
	if strings.HasPrefix(AppConfig.AbsolutePath, AppConfig.UserHomeDir) && AppConfig.Username != "root" {
		AppConfig.DisplayCdPath = strings.Replace(AppConfig.DisplayCdPath, AppConfig.UserHomeDir, "~", 1)
	}
}

var (
	builtins = map[string]BuiltinFunc{
		"exit": exit,
		"cd":   cd,
		"pwd":  pwd,
	}
)

func findCommand(command string) (BuiltinFunc, error) {
	if val, ok := builtins[command]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("command not found: %s", command)
}

// $-- BUILT-INS --

type Handler = func(*Runner, io.Writer, io.Writer) error

type Runner struct {
	tokenizer    Tokenizer
	chain        ExpansionChain
	pathSearcher PathSearcher
}

func (r *Runner) init() {
	r.tokenizer = NewTokenizer()
	r.chain = NewExpansionChain()
	r.pathSearcher = NewPathSearcher()
}

func NewRunner() Runner {
	res := Runner{}
	res.init()
	return res
}

func (r *Runner) Clear() {
	r.tokenizer.Clear()
}

// RunCommand Used for interactive execution and execution with -c flag
// TODO -c flag
func (r *Runner) RunCommand(commands []string, stdout io.Writer, stderr io.Writer) error {
	if commands == nil || len(commands) == 0 {
		return nil
	}

	cmd, err := findCommand(commands[0])
	if err == nil {
		if err := cmd(commands[1:]...); err != nil {
			fmt.Println(stderr, err)
		}
		return nil
	}
	binary, err2 := r.pathSearcher.FindBinary(commands[0])
	if err2 != nil {
		_, err := fmt.Fprintln(stderr, err2)
		if err != nil {
			return err
		}
	} else {
		var outbuf, errbuf strings.Builder
		ex := exec.Command(binary, commands[1:]...)
		ex.Stdout = &outbuf
		ex.Stderr = &errbuf
		err := ex.Run()
		if err != nil {
			_, err = fmt.Fprintf(stderr, errbuf.String())
			if err != nil {
				return err
			}
		}
		_, err = fmt.Fprintf(stdout, outbuf.String())
		if err != nil {
			return err
		}
	}
	return nil
}

func DefaultHandler(r *Runner, stdout io.Writer, stderr io.Writer) error {
	tokens := r.tokenizer.CollectTokens()
	tokens = r.chain.Execute(tokens)
	// -- TODO remove --
	_, err := fmt.Fprintf(stdout, "%v", tokens)
	if err != nil {
		return err
	}
	// $-- TODO --
	commands := ToCommand(tokens)
	err = r.RunCommand(commands, stdout, stderr)
	if err != nil {
		return err
	}
	return nil
}

func EmptyHandler(r *Runner, stdout io.Writer, stderr io.Writer) error {
	return nil
}

func (r *Runner) RunInteractive(stdin io.Reader, stdout io.Writer, stderr io.Writer, runnerCall Handler) error {

	reader := bufio.NewReader(stdin)
	isEof := false
	for !isEof {
		_, err := fmt.Fprintf(stdout, "%s%s ", AppConfig.DisplayCdPath, AppConfig.Prompt)
		if err != nil {
			return err
		}
		text, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				isEof = true
			} else {
				return err
			}
		}
		text = strings.Trim(text, " \n\t")
		r.tokenizer.Tokenize(text)
		for !r.tokenizer.IsComplete() && !isEof {
			_, err = fmt.Fprint(stdout, "> ")
			if err != nil {
				return err
			}
			text, err = reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					isEof = true
				} else {
					return err
				}
			}
			r.tokenizer.Tokenize(text)
		}
		err = runnerCall(r, stdout, stderr)
		if err != nil {
			return err
		}
	}
	return nil
}

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

func findBuiltIn(command string) (BuiltinFunc, error) {
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
func (r *Runner) RunCommand(commands [][]string, stdout io.Writer, stderr io.Writer) error {
	if commands == nil || len(commands) == 0 {
		return nil
	}
	if len(commands) == 1 {
		// Due to the fact that pipes run in individual
		// subshells and state of app remains untouched
		// we will not allow usage of builtIns in pipes (for now)
		// it'll be pain in the ass to refactor again
		// TODO make custom stdin, stdout and stderr for builtIns

		res := strings.SplitN(commands[0][0], "=", 2)
		if len(res) == 2 {
			err := os.Setenv(res[0], res[1])
			if err != nil {
				return err
			}
			return nil
		}

		builtIn, err := findBuiltIn(commands[0][0])
		if err == nil {
			if err := builtIn(commands[0][1:]...); err != nil {
				fmt.Println(stderr, err)
			}
			return nil
		}
	}
	var execs []*exec.Cmd
	for i, command := range commands {
		execs = append(execs,
			exec.Command(command[0], command[1:]...))
		if i > 0 {
			pipe := execs[i]
			pipe.Stdin, _ = execs[i-1].StdoutPipe()
		}
	}
	var outbuf, errbuf strings.Builder
	execs[len(execs)-1].Stdout = &outbuf
	execs[len(execs)-1].Stderr = &errbuf

	var err error
	if len(execs) == 1 {
		err = execs[0].Run()
	} else {
		c2 := execs[len(execs)-1]
		c1 := execs[len(execs)-2]
		for i := len(execs) - 2; i >= 0; i-- {
			c1 = execs[i]
			err = c2.Start()
			if err != nil {
				break
			}
			err = c1.Run()
			if err != nil {
				break
			}
			err = c2.Wait()
			if err != nil {
				break
			}
			c2 = c1
		}
	}
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
	pipedCommands := splitInPipes(tokens)
	var commands [][]string
	for _, command := range pipedCommands {
		commands = append(commands, ToCommand(command))
	}
	err = r.RunCommand(commands, stdout, stderr)
	if err != nil {
		return err
	}
	return nil
}

func splitInPipes(tokens []Token) [][]Token {
	var res [][]Token
	var tmp []Token
	for _, token := range tokens {
		if token == PipeTokenInstance && len(tmp) > 0 {
			res = append(res, tmp)
			tmp = nil
		} else {
			tmp = append(tmp, token)
		}
	}
	if len(tmp) > 0 {
		res = append(res, tmp)
		tmp = nil
	}

	return res
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
		if shErr := r.tokenizer.Tokenize(text); shErr != nil {
			_, err = fmt.Fprintln(stderr, shErr.Error())
			if err != nil {
				return err
			}
		}
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
			if shErr := r.tokenizer.Tokenize(text); shErr != nil {
				_, err = fmt.Fprintln(stderr, shErr.Error())
				if err != nil {
					return err
				}
			}
		}
		err = runnerCall(r, stdout, stderr)
		if err != nil {
			return err
		}
	}
	return nil
}

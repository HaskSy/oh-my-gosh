package main

import (
	"bufio"
	"fmt"
	. "gosh/pathSearcher"
	shErr "gosh/shellError"
	. "gosh/tokenizer"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// -- CONFIGURATION --
var (
	shellName     = "gosh"
	prompt        = "$"
	displayCdPath = ""
	absolutePath  = ""
	userHomeDir   = ""
	username      = ""
)

// $-- CONFIGURATION --$

// -- EXPANSION CHAIN --

type ExpansionRule = func(token Token) Token

type ExpansionChain struct {
	lastChange    []Token
	expansionList []ExpansionRule
}

func NewExpansionChain(exs ...ExpansionRule) ExpansionChain {
	return ExpansionChain{nil, exs}
}

func (ec *ExpansionChain) Append(ex ExpansionRule) *ExpansionChain {
	ec.expansionList = append(ec.expansionList, ex)
	return ec
}

func (ec *ExpansionChain) Execute(tokens []Token) []Token {
	ec.lastChange = tokens
	for _, exp := range ec.expansionList {
		var curr []Token
		for _, token := range ec.lastChange {
			curr = append(curr, exp(token))
		}
		ec.lastChange = curr
	}
	return ec.lastChange
}

func tildaExpansion(token Token) Token {
	newToken := token
	if token.TokenType == WordToken {
		if strings.HasPrefix(token.Value, "~/") {
			newToken.Value = filepath.Join(userHomeDir, token.Value[2:])
		} else if token.Value == "~" {
			newToken.Value = userHomeDir
		}
	}
	return newToken
}

func variableExpansion(token Token) Token {
	newToken := token
	if token.TokenType == WordToken {
		if strings.HasPrefix(token.Value, "$") {
			newToken.Value = os.Getenv(token.Value[1:])
		}
	}
	return newToken
}

// $-- EXPANSION CHAIN --$

// -- BUILT-INS --

type BuiltinFunc = func(...string) shErr.ShellError

func exit(args ...string) shErr.ShellError {
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

func cd(args ...string) shErr.ShellError {
	var directory string
	if len(args) > 1 {
		return shErr.New("cd", "too many arguments")
	} else if len(args) == 1 && args[0] != "~" {
		directory = args[0]
	} else {
		directory = userHomeDir
	}

	var err error
	if err = syscall.Chdir(directory); err != nil {
		return shErr.NewWithBlame("cd", "No such file or directory", directory)
	}
	absolutePath, err = os.Getwd()
	tildifyCdPath()
	return nil
}

func pwd(_ ...string) shErr.ShellError {
	fmt.Println(absolutePath)
	return nil
}

func tildifyCdPath() {
	displayCdPath = absolutePath
	if strings.HasPrefix(absolutePath, userHomeDir) && username != "root" {
		displayCdPath = strings.Replace(displayCdPath, userHomeDir, "~", 1)
	}
}

var (
	builtins = map[string]BuiltinFunc{
		"exit": exit,
		"cd":   cd,
		"pwd":  pwd,
	}
)

// $-- BUILT-INS --$

func findCommand(command string) (BuiltinFunc, shErr.ShellError) {
	if val, ok := builtins[command]; ok {
		return val, nil
	}
	return nil, shErr.New(command, "command not found")
}

// -- State --
var (
	tokenizer    = NewTokenizer()
	pathSearcher = NewPathSearcher()
	chain        = NewExpansionChain(
		tildaExpansion,
		//variableExpansion,
	)
)

// $-- State --$

func init() {
	curr, err := user.Current()
	if err != nil {
		panic(err)
	}
	userHomeDir = curr.HomeDir
	username = curr.Username

	if username == "root" {
		prompt = "#"
	}

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	absolutePath = filepath.Dir(ex)
	tildifyCdPath()
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s%s ", displayCdPath, prompt)
		text, _ := reader.ReadString('\n')
		text = strings.Trim(text, " \n\t")
		tokenizer.Tokenize(text)
		for !tokenizer.IsComplete() {
			fmt.Print("> ")
			text, _ = reader.ReadString('\n')
			tokenizer.Tokenize(text)
		}
		tokens := tokenizer.CollectTokens()
		tokens = chain.Execute(tokens)
		commands := ToCommand(tokens)
		cmd, err := findCommand(commands[0])
		if err != nil {
			binary, err := pathSearcher.FindBinary(commands[0])
			if err != nil {
				fmt.Println(err)
			} else {
				var outbuf, errbuf strings.Builder
				ex := exec.Command(binary, commands[1:]...)
				ex.Stdout = &outbuf
				ex.Stderr = &errbuf
				err := ex.Run()
				if err != nil {
					fmt.Printf(errbuf.String())
				}
				fmt.Printf(outbuf.String())
			}
		} else {
			if err := cmd(commands[1:]...); err != nil {
				fmt.Println(err)
			}
		}
	}
}

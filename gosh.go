package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unicode"
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

// -- TOKEN & PARSE --
type TokenType = int

const (
	WordToken TokenType = iota
	WeakQuotationToken
	StrongQuotationToken
	SpecialCharToken
	PipeToken
)

type Token struct {
	tokenType TokenType
	value     string
}

func Tokenize(command string) []Token {
	var tokens []Token
	var currentToken string
	var currentTokenType TokenType
	var prevTokenType TokenType
	var consecBackslash int
	for _, char := range command {
		if char == '\\' {
			consecBackslash++
			continue
		}
		switch {
		case unicode.IsSpace(char):
			if currentToken != "" {
				tokens = append(tokens, Token{currentTokenType, currentToken})
				currentToken = ""
			}
			currentTokenType = WordToken
		case currentTokenType == SpecialCharToken:
			// TODO: check whether given char is actually referred to some special character (depending on the quotation)
			tokens = append(tokens, Token{currentTokenType, string(char)})
			currentToken = ""
			currentTokenType = prevTokenType
		case currentTokenType == StrongQuotationToken:
		case currentTokenType == WeakQuotationToken:
		}
		consecBackslash = 0
	}
	return tokens
}

// $-- TOKEN & PARSE --$

// -- EXPANSION CHAIN --

type BuiltinFunc = func(...string) error

type Expansion = func(string) string

type ExpansionChain struct {
	lastChange    string
	expansionList []Expansion
}

func (ec *ExpansionChain) Append(ex Expansion) *ExpansionChain {
	ec.expansionList = append(ec.expansionList, ex)
	return ec
}

func (ec *ExpansionChain) Execute(str string) string {
	ec.lastChange = str
	for _, exp := range ec.expansionList {
		ec.lastChange = exp(ec.lastChange)
	}
	return ec.lastChange
}

// $-- EXPANSION CHAIN --$

// -- BUILT-INS --

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
		return errors.New(fmt.Sprintf("%s: too many arguments", "cd"))
	} else if len(args) == 1 && args[0] != "~" {
		directory = args[0]
	} else {
		directory = userHomeDir
	}

	var err error
	if err = syscall.Chdir(directory); err != nil {
		return errors.New(fmt.Sprintf("%s: %s: No such file or directory", "cd", directory))
	}
	absolutePath, err = os.Getwd()
	tildifyCdPath()
	return nil
}

func pwd(args ...string) error {
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

func findCommand(command string) (BuiltinFunc, error) {
	if val, ok := builtins[command]; ok {
		return val, nil
	}
	return nil, errors.New(fmt.Sprintf("%s: command not found", command))
}

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
		text, _, _ := reader.ReadLine()
		commands := strings.Split(string(text), " ")
		cmd, err := findCommand(commands[0])
		if err != nil {
			fmt.Printf("%s: %s\n", shellName, err)
		} else {
			if err := cmd(commands[1:]...); err != nil {
				fmt.Printf("%s: %s\n", shellName, err)
			}
		}
	}
}

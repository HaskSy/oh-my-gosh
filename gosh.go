package main

import (
	"bufio"
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

// -- EXCEPTIONS --
type shellErrorObj struct {
	command string
	msg     string
	blame   *string
}

type ShellError = *shellErrorObj

func (e ShellError) Error() string {
	if e.blame == nil {
		return fmt.Sprintf("%s: %s: %s", shellName, e.command, e.msg)
	}
	return fmt.Sprintf("%s: %s: %s: %s", shellName, e.command, *e.blame, e.msg)
}

func NewWithBlame(command string, msg string, blame string) ShellError {
	return &shellErrorObj{command: command, msg: msg, blame: &blame}
}

func New(command string, msg string) ShellError {
	return &shellErrorObj{command: command, msg: msg, blame: nil}
}

// $-- EXCEPTIONS --$

// -- TOKENIZER --

type TokenType = int

const (
	WordToken TokenType = iota

	// -- Quotations --

	WeakQuotationToken
	StrongQuotationToken

	// $-- Quotations --$

	PipeToken
)

type Token struct {
	tokenType TokenType
	value     string
}

type Tokenizer struct {
	currentToken     string
	currentTokenType TokenType
	consecBackslash  int
	tokens           []Token
}

func (t *Tokenizer) init() {
	t.currentTokenType = WordToken
	t.tokens = []Token{}
}

func NewTokenizer() Tokenizer {
	res := Tokenizer{}
	res.init()
	return res
}

func (t *Tokenizer) isQuoted() bool {
	return t.currentTokenType >= WeakQuotationToken && t.currentTokenType <= StrongQuotationToken
}

func (t *Tokenizer) isWeakQuoted() bool {
	return t.currentTokenType == WeakQuotationToken
}

func (t *Tokenizer) isStrongQuoted() bool {
	return t.currentTokenType == StrongQuotationToken
}

func (t *Tokenizer) appendToken() {
	if t.currentToken != "" {
		t.tokens = append(t.tokens, Token{t.currentTokenType, t.currentToken})
		t.currentToken = ""
	}
	t.currentTokenType = WordToken
}

func (t *Tokenizer) expectsPipe() bool {
	return len(t.tokens) > 0 && t.tokens[len(t.tokens)-1].tokenType == PipeToken
}

func (t *Tokenizer) IsComplete() bool {
	return t.currentToken == "" &&
		t.currentTokenType == WordToken && !t.expectsPipe() && t.consecBackslash%2 == 0
}

// CollectTokens Get tokens and flush state
func (t *Tokenizer) CollectTokens() []Token {
	if t.IsComplete() {
		res := t.tokens
		t.init()
		return res
	}
	return nil
}

func (t *Tokenizer) Tokenize(command string) {
	if !t.IsComplete() {
		t.currentToken += "\n"
	}

	for _, char := range command {
		if char == '\\' {
			t.currentToken += string(char)
			t.consecBackslash++
			continue
		}
		switch {
		case unicode.IsSpace(char) && !t.isQuoted():
			t.appendToken()
		case char == '\'':
			if t.isWeakQuoted() {
				t.currentToken += string(char)
				t.appendToken()
			} else if t.isStrongQuoted() {
				t.currentToken += string(char)
			} else {
				t.appendToken()
				t.currentToken += string(char)
				t.currentTokenType = WeakQuotationToken
			}
		case char == '"':
			if !t.isQuoted() {
				t.appendToken()
				t.currentToken += string(char)
				t.currentTokenType = StrongQuotationToken
			} else {
				t.currentToken += string(char)
				if t.isStrongQuoted() && t.consecBackslash%2 == 0 {
					t.appendToken()
				}
			}
		case char == '|':
			if !t.isQuoted() {
				t.appendToken()
				t.currentToken += string(char)
				t.currentTokenType = PipeToken
				t.appendToken()
			} else {
				t.currentToken += string(char)
			}
		default:
			t.currentToken += string(char)
		}
		t.consecBackslash = 0
	}
	if t.currentTokenType == WordToken {
		t.appendToken()
	}
}

// $-- TOKENIZER --$

// -- EXPANSION CHAIN --

type BuiltinFunc = func(...string) ShellError

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

func exit(args ...string) ShellError {
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

func cd(args ...string) ShellError {
	var directory string
	if len(args) > 1 {
		return New("cd", "too many arguments")
	} else if len(args) == 1 && args[0] != "~" {
		directory = args[0]
	} else {
		directory = userHomeDir
	}

	var err error
	if err = syscall.Chdir(directory); err != nil {
		return NewWithBlame("cd", "No such file or directory", directory)
	}
	absolutePath, err = os.Getwd()
	tildifyCdPath()
	return nil
}

func pwd(_ ...string) ShellError {
	fmt.Println(absolutePath)
	return nil
}

func clear(_ ...string) ShellError {
	fmt.Printf("\033[2J")
	fmt.Printf("\033[H")
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
		"exit":  exit,
		"cd":    cd,
		"pwd":   pwd,
		"clear": clear,
	}
)

// $-- BUILT-INS --$

func findCommand(command string) (BuiltinFunc, ShellError) {
	if val, ok := builtins[command]; ok {
		return val, nil
	}
	return nil, New(command, "command not found")
}

// -- State --
var (
	tokenizer = NewTokenizer()
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
		text = strings.Trim(text, " \n")
		tokenizer.Tokenize(text)
		for !tokenizer.IsComplete() {
			fmt.Print("> ")
			text, _ = reader.ReadString('\n')
			tokenizer.Tokenize(text)
		}
		tokens := tokenizer.CollectTokens()
		fmt.Printf("%v", tokens)
		commands := strings.Split(text, " ")
		cmd, err := findCommand(commands[0])
		if err != nil {
			fmt.Println(err)
		} else {
			if err := cmd(commands[1:]...); err != nil {
				fmt.Println(err)
			}
		}
	}
}

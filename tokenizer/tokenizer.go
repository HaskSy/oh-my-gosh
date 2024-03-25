package tokenizer

import (
	"fmt"
	"gosh/shellError"
	"regexp"
	"strings"
	"unicode"
)

type TokenType = int

const (
	WordToken TokenType = iota

	// -- Quotations --

	WeakQuotationToken
	StrongQuotationToken

	// $-- Quotations --$

	SpaceToken
	PipeToken
	DollarToken
	AssignToken
)

type Token struct {
	TokenType TokenType
	Value     string
}

// All the following tokens are identical, so we're creating single object for each one to save up space.
// Golang doesn't allow immutable structures, so DON'T MUTATE THIS STRUCTURE

var SpaceTokenInstance = Token{SpaceToken, " "}
var PipeTokenInstance = Token{PipeToken, "|"}
var DollarTokenInstance = Token{DollarToken, "$"}
var AssignTokenInstance = Token{AssignToken, "="}

type Tokenizer struct {
	currentToken     string
	assignVarToken   string // Keeps the variable name for token assign to then store it like `(=) varname value`
	currentTokenType TokenType
	consecBackslash  int
	expectNewline    bool
	expectSpaceToken bool
	isFirstLine      bool
	tokens           []Token
}

func (t *Tokenizer) init() {
	t.currentTokenType = WordToken
	t.tokens = []Token{}
	t.isFirstLine = true
}

func (t *Tokenizer) Clear() {
	t.currentToken = ""
	t.currentTokenType = WordToken
	t.consecBackslash = 0
	t.tokens = []Token{}
	t.expectNewline = false
	t.expectSpaceToken = false
	t.expectSpaceToken = true
}

func NewTokenizer() Tokenizer {
	res := Tokenizer{}
	res.init()
	return res
}

func (t *Tokenizer) isQuoted() bool {
	return t.currentTokenType == WeakQuotationToken || t.currentTokenType == StrongQuotationToken
}

func (t *Tokenizer) isWeakQuoted() bool {
	return t.currentTokenType == WeakQuotationToken
}

func (t *Tokenizer) isStrongQuoted() bool {
	return t.currentTokenType == StrongQuotationToken
}

func (t *Tokenizer) appendToken() {
	t.addBackslashes()
	if t.currentToken != "" {
		t.addSpaceToken()
		if len(t.tokens) > 0 && t.currentTokenType == WordToken && t.tokens[len(t.tokens)-1].TokenType == DollarToken {
			varname, command := SplitInVarTokens(t.currentToken)
			if varname != "" {
				t.tokens = append(t.tokens, Token{t.currentTokenType, varname})
			}
			if command != "" {
				t.tokens = append(t.tokens, Token{t.currentTokenType, command})
			}
		} else {
			t.tokens = append(t.tokens, Token{t.currentTokenType, t.currentToken})
		}
		t.currentToken = ""
	}
	t.currentTokenType = WordToken
}

func SplitInVarTokens(s string) (string, string) {
	if unicode.IsDigit(rune(s[0])) {
		return string(s[0]), s[1:]
	}
	if !((s[0] >= 'a' && s[0] <= 'z') ||
		(s[0] >= 'A' && s[0] <= 'Z') ||
		(s[0] == '_')) {
		return "", s
	}
	pattern := "^[a-zA-Z_][a-zA-Z0-9_]*"
	regex := regexp.MustCompile(pattern)
	index := regex.FindStringIndex(s)
	if index != nil && index[0] == 0 {
		return s[:index[1]], s[index[1]:]
	}
	return "", s
}

func (t *Tokenizer) addSpaceToken() {
	if !t.expectSpaceToken || len(t.tokens) == 0 {
		t.expectSpaceToken = false
		return
	}
	if lastToken := t.tokens[len(t.tokens)-1]; lastToken != SpaceTokenInstance &&
		lastToken != PipeTokenInstance {
		t.tokens = append(t.tokens, SpaceTokenInstance)
		t.expectSpaceToken = false
	}
}

func (t *Tokenizer) addPipeToken() shellError.ShellError {
	if len(t.tokens) == 0 {
		return shellError.New(fmt.Sprintf("syntax error near unexpected token `|'"))
	}
	if len(t.tokens) > 0 && t.tokens[len(t.tokens)-1] == SpaceTokenInstance {
		t.tokens[len(t.tokens)-1] = PipeTokenInstance
	} else {
		t.tokens = append(t.tokens, PipeTokenInstance)
	}
	return nil
}

func (t *Tokenizer) addDollarToken() {
	t.tokens = append(t.tokens, DollarTokenInstance)
}

func (t *Tokenizer) addAssignToken() {
	t.tokens = append(t.tokens, AssignTokenInstance)
}

func (t *Tokenizer) addBackslashes() {
	t.currentToken += strings.Repeat("\\", t.consecBackslash)
	t.consecBackslash = 0
}

func (t *Tokenizer) addCharacter(r rune) {
	t.addBackslashes()
	t.currentToken += string(r)
}

func (t *Tokenizer) expectsPipe() bool {
	return len(t.tokens) > 0 && t.tokens[len(t.tokens)-1] == PipeTokenInstance
}

func (t *Tokenizer) IsComplete() bool {
	return t.currentToken == "" &&
		t.currentTokenType == WordToken && !t.expectsPipe() && t.consecBackslash%2 == 0 && !t.expectNewline
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

// ToCommand Remove space tokens and merge tokens not separated by space
func ToCommand(tokens []Token) []string {
	var newTokens []string
	tmp := Token{}
	tmp.TokenType = WordToken
	for _, token := range tokens {
		if token.TokenType == SpaceToken {
			newTokens = append(newTokens, tmp.Value)
			tmp.Value = ""
		} else {
			tmp.Value = tmp.Value + token.Value
		}
	}
	if tmp.Value != "" {
		newTokens = append(newTokens, tmp.Value)
	}
	return newTokens
}

// Tokenize accepts new line of input and updates state of a tokenizer.
// is not expected to be used individually. Import is expected to be trimmed
// depending on the current state of the shell
func (t *Tokenizer) Tokenize(command string) shellError.ShellError {
	t.consecBackslash = 0 // multiline command
	for _, char := range command {
		if char == '\\' {
			// backslashes are processed individually later
			t.consecBackslash++
			continue
		}
		if unicode.IsSpace(char) {
			// generalizes handling of <newline>, <tab> and <space> as separators
			if t.isQuoted() {
				t.addCharacter(char)
			} else {
				if t.currentToken != "" {
					t.appendToken()
				} else if t.consecBackslash > 0 {
					t.consecBackslash -= t.consecBackslash % 2
					t.appendToken()
					t.expectNewline = true
				}
				t.expectSpaceToken = true
			}
			continue
		}
		t.expectNewline = false
		switch {
		case char == '\'':
			if t.isWeakQuoted() {
				// finishes Weak Quotation (next WQ). Bash doesn't allow for escape characters in weak quotations
				t.appendToken()
			} else if t.isStrongQuoted() || t.consecBackslash%2 == 1 {
				// Character in SQ and escape characters in default
				t.addCharacter(char)
			} else {
				// NewCommand WQ initialized
				t.appendToken()
				t.currentTokenType = WeakQuotationToken
			}
		case char == '"':
			if !t.isQuoted() && t.consecBackslash%2 == 0 {
				// NewCommand SQ if no other quotation is present and not an escape character in default
				t.appendToken()
				t.currentTokenType = StrongQuotationToken
			} else if t.isStrongQuoted() && t.consecBackslash%2 == 0 {
				// If part of SQ and not an escape character
				t.appendToken()
			} else {
				// Escape character
				t.addCharacter(char)
			}
		case char == '|':
			if t.isQuoted() || t.consecBackslash%2 == 1 {
				t.addCharacter(char)
			} else {
				t.appendToken()
				err := t.addPipeToken()
				if err != nil {
					return err
				}
			}
		case char == '$':
			if t.isQuoted() || t.consecBackslash%2 == 1 {
				t.addCharacter(char)
			} else {
				t.addSpaceToken()
				t.appendToken()
				t.addDollarToken()
			}
		case char == '=':
			if t.isQuoted() || t.consecBackslash%2 == 1 {
				t.addCharacter(char)
			} else {
				t.addSpaceToken()
				t.appendToken()
				last := t.tokens[len(t.tokens)-1].Value
				varname, _ := SplitInVarTokens(last)
				if varname != last {
					return shellError.NewCommand("=", fmt.Sprintf("Invalid env variable name: %s", varname))
				}
				t.addAssignToken()
			}
		default:
			// Handling the rest
			t.addCharacter(char)
		}
	}
	if t.currentTokenType == WordToken && t.consecBackslash%2 == 0 {
		t.appendToken()
	} else if t.isFirstLine &&
		(t.currentTokenType == WeakQuotationToken || t.currentTokenType == StrongQuotationToken) {
		t.addCharacter('\n')
	}
	t.isFirstLine = false
	return nil
}

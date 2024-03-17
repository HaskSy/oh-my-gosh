package tokenizer

import (
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

	PipeToken
	SpaceToken
)

type Token struct {
	TokenType TokenType
	Value     string
}

// All the following tokens are identical, so we're creating single object for each one to save up space.
// Golang doesn't allow immutable structures, so DON'T MUTATE THIS STRUCTURE
var SpaceTokenInstance = Token{SpaceToken, " "}
var PipeTokenInstance = Token{PipeToken, ""}

type Tokenizer struct {
	currentToken     string
	currentTokenType TokenType
	consecBackslash  int
	expectNewline    bool
	tokens           []Token
}

func (t *Tokenizer) init() {
	t.currentTokenType = WordToken
	t.tokens = []Token{}
	t.expectNewline = false
}

func (t *Tokenizer) Clear() {
	t.currentToken = ""
	t.currentTokenType = WordToken
	t.consecBackslash = 0
	t.tokens = []Token{}
	t.expectNewline = false
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

func (t *Tokenizer) appendWithSpaceToken(forceSpace bool) {
	if t.currentToken != "" {
		t.addBackslashes()
		t.tokens = append(t.tokens, Token{t.currentTokenType, t.currentToken})
		t.currentToken = ""
		t.tokens = append(t.tokens, SpaceTokenInstance)
	} else if forceSpace {
		t.addSpaceToken()
	}
	t.currentTokenType = WordToken
}

func (t *Tokenizer) addPipeToken() {
	if t.tokens[len(t.tokens)-1] == SpaceTokenInstance {
		t.tokens[len(t.tokens)-1] = PipeTokenInstance
	} else {
		t.tokens = append(t.tokens, PipeTokenInstance)
	}
}

func (t *Tokenizer) addSpaceToken() {
	if t.tokens[len(t.tokens)-1] != SpaceTokenInstance {
		t.tokens = append(t.tokens, SpaceTokenInstance)
	}
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
func (t *Tokenizer) Tokenize(command string) {
	t.consecBackslash = 0 // multiline command
	for _, char := range command {
		if char == '\\' {
			// backslashes are processed individually later
			t.consecBackslash++
			continue
		}
		if unicode.IsSpace(char) && !t.isQuoted() {
			// generalizes handling of <newline>, <tab> and <space> as separators
			if t.currentToken == "" &&
				t.consecBackslash > 0 {

				trail := t.consecBackslash % 2
				t.consecBackslash -= trail
				t.addBackslashes()
				t.appendWithSpaceToken(true)
				t.expectNewline = true
			} else if t.consecBackslash%2 == 0 {
				t.appendWithSpaceToken(false)
			}
			continue
		}
		t.expectNewline = false
		switch {
		case char == '\'':
			if t.isWeakQuoted() {
				// finishes Weak Quotation (next WQ). Bash doesn't allow for escape characters in weak quotations
				t.addBackslashes()
				t.appendToken()
			} else if t.isStrongQuoted() || t.consecBackslash%2 == 1 {
				// Character in SQ and escape characters in default
				t.addCharacter(char)
			} else {
				// New WQ initialized
				t.appendToken()
				t.currentTokenType = WeakQuotationToken
			}
		case char == '"':
			if !t.isQuoted() && t.consecBackslash%2 == 0 {
				// New SQ if no other quotation is present and not an escape character in default
				t.addBackslashes()
				t.appendToken()
				t.currentTokenType = StrongQuotationToken
			} else if t.isStrongQuoted() && t.consecBackslash%2 == 0 {
				// If part of SQ and not an escape character
				t.addBackslashes()
				t.appendToken()
			} else {
				// Escape character
				t.addCharacter(char)
			}
		case char == '|':
			if t.isQuoted() {
				// Pipes cannot appear in quotations
				t.addCharacter(char)
			} else {
				t.appendToken()
				t.addPipeToken()
			}
		default:
			// Handling the rest
			t.addCharacter(char)
		}
	}
	if t.currentTokenType == WordToken && t.consecBackslash%2 == 0 {
		t.addBackslashes()
		t.appendToken()
	}
}

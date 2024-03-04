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

func (t *Tokenizer) appendWithSpaceToken() {
	if t.currentToken != "" {
		t.tokens = append(t.tokens, Token{t.currentTokenType, t.currentToken})
		t.currentToken = ""
		t.tokens = append(t.tokens, Token{SpaceToken, " "})
	}
	t.currentTokenType = WordToken
}

func (t *Tokenizer) addPipeToken() {
	t.tokens = append(t.tokens, Token{PipeToken, ""})
}

func (t *Tokenizer) addBackslashes() {
	t.currentToken += strings.Repeat("\\", t.consecBackslash-t.consecBackslash%2)
	t.consecBackslash = t.consecBackslash % 2
}

func (t *Tokenizer) addCharacter(r rune) {
	t.addBackslashes()
	t.currentToken += string(r)
}

func (t *Tokenizer) expectsPipe() bool {
	return len(t.tokens) > 0 && t.tokens[len(t.tokens)-1].TokenType == PipeToken
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

// ToCommand Remove space tokens and merge tokens not splitted by space
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

func (t *Tokenizer) Tokenize(command string) {
	t.consecBackslash = 0
	for _, char := range command {
		if char == '\\' {
			t.consecBackslash++
			continue
		}
		if unicode.IsSpace(char) && !t.isQuoted() {
			if t.consecBackslash%2 == 0 {
				t.appendWithSpaceToken()
			}
			continue
		}
		if t.consecBackslash == 1 && t.isQuoted() {
			t.addCharacter('\\')
		}
		switch {
		case char == '\'':
			if t.isWeakQuoted() {
				t.appendToken()
			} else if t.isStrongQuoted() {
				t.addCharacter(char)
			} else {
				t.appendToken()
				t.currentTokenType = WeakQuotationToken
			}
		case char == '"':
			if !t.isQuoted() {
				t.appendToken()
				t.currentTokenType = StrongQuotationToken
			} else if t.isStrongQuoted() && t.consecBackslash%2 == 0 {
				t.appendToken()
			} else {
				t.addCharacter(char)
			}
		case char == '|':
			if !t.isQuoted() {
				t.appendToken()
				t.addPipeToken()
			} else {
				t.addCharacter(char)
			}
		default:
			t.addCharacter(char)
		}
		t.consecBackslash = 0
	}
	if t.currentTokenType == WordToken && t.consecBackslash%2 == 0 {
		t.addBackslashes()
		t.appendToken()
	}
}

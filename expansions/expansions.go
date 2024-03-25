package expansions

import (
	. "gosh/config"
	. "gosh/tokenizer"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// -- EXPANSION CHAIN --

type ExpansionRule = func(tokens []Token) []Token

type ExpansionChain struct {
	lastChange    []Token
	expansionList []ExpansionRule
}

func NewExpansionChain() ExpansionChain {
	exs := []ExpansionRule{
		tildaExpansion,
		variableExpansion,
		escapingCharacters,
	}
	return ExpansionChain{nil, exs}
}

func (ec *ExpansionChain) Append(ex ExpansionRule) *ExpansionChain {
	ec.expansionList = append(ec.expansionList, ex)
	return ec
}

// Execute applies all expansion rules to the provided list of tokens.
func (ec *ExpansionChain) Execute(tokens []Token) []Token {
	ec.lastChange = tokens
	for _, exp := range ec.expansionList {
		ec.lastChange = exp(ec.lastChange)
	}
	return ec.lastChange
}

// tildaExpansion expands tilde (~) to user's home directory.
func tildaExpansion(tokens []Token) []Token {
	for i, token := range tokens {
		if token.TokenType == WordToken {
			if token.Value == "~" {
				tokens[i].Value = AppConfig.UserHomeDir
			} else if strings.HasPrefix(token.Value, "~/") {
				tokens[i].Value = filepath.Join(AppConfig.UserHomeDir, token.Value[2:])
			}
		}
	}
	return tokens
}

// variableExpansion expands variables to their values if valid.
func variableExpansion(tokens []Token) []Token {
	var newTokens []Token
	for _, token := range tokens {
		if ln := len(newTokens); ln > 0 && token.TokenType == WordToken && newTokens[ln-1].TokenType == DollarToken {
			lastToken := newTokens[ln-1]
			if isValidVarName(token.Value) {
				lastToken.Value = os.Getenv(token.Value)
			} else {
				lastToken.Value += token.Value
			}
			lastToken.TokenType = WordToken
			newTokens = append(newTokens[:ln-1], lastToken)
		} else if token.TokenType == StrongQuotationToken {
			token.Value = expandStrongQuotation(token.Value)
			newTokens = append(newTokens, token)
		} else {
			newTokens = append(newTokens, token)
		}
	}
	return newTokens
}

func isValidVarName(s string) bool {
	if len(s) == 1 && unicode.IsDigit(rune(s[0])) {
		return true
	}
	pattern := "[a-zA-Z_][a-zA-Z0-9_]*$"
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(s)
}

func expandStrongQuotation(s string) string {
	newString := ""
	for len(s) > 0 {
		isDone := true
		for i, c := range s {
			if c == '$' {
				token, remaining := SplitInVarTokens(s[i+1:])
				if token != "" {
					newString += os.Getenv(token)
					s = remaining
				} else {
					newString += remaining
					s = ""
				}
				isDone = false
				break
			} else {
				newString += string(c)
			}
		}
		if isDone {
			break
		}
	}
	return newString
}

func escapingCharacters(tokens []Token) []Token {
	var newTokens []Token
	for _, token := range tokens {
		if token.TokenType == WordToken || token.TokenType == StrongQuotationToken {
			token.Value = escapingCharactersString(token.Value)
		}
		newTokens = append(newTokens, token)
	}
	return newTokens

}

func escapingCharactersString(str string) string {
	newString := ""
	isEscape := false
	for _, c := range str {
		if c == '\\' && !isEscape {
			isEscape = true
			continue
		}
		newString += string(c)
		isEscape = false
	}
	return newString
}

// $-- EXPANSION CHAIN --

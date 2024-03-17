package expansions

import (
	. "gosh/config"
	. "gosh/tokenizer"
	"os"
	"path/filepath"
	"strings"
)

// -- EXPANSION CHAIN --

type ExpansionRule = func(token Token) Token

type ExpansionChain struct {
	lastChange    []Token
	expansionList []ExpansionRule
}

func NewExpansionChain() ExpansionChain {
	exs := []ExpansionRule{
		tildaExpansion,
		variableExpansion,
	}
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
			newToken.Value = filepath.Join(AppConfig.UserHomeDir, token.Value[2:])
		} else if token.Value == "~" {
			newToken.Value = AppConfig.UserHomeDir
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

// $-- EXPANSION CHAIN --

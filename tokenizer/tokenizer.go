package tokenizer

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
		//case unicode.IsSpace(char) && !t.isQuoted():
		//	t.appendToken()
		case char == '\'':
			if t.isWeakQuoted() {
				t.appendToken()
			} else if t.isStrongQuoted() {
				t.currentToken += string(char)
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
				t.currentToken += string(char)
			}
		case char == '|':
			if !t.isQuoted() {
				t.appendToken()
				t.currentToken += string(char) // TODO: Is it actually necessary?
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

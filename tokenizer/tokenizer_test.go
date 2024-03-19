package tokenizer

import (
	"testing"
)

var tokenizer Tokenizer

func beforeEach() {
	tokenizer.Clear()
}

func afterAll() {
	tokenizer.Clear()
}

func executeTokenizerTests(t *testing.T, tests []Test) {
	for testId, test := range tests {
		beforeEach()
		for _, p := range test.Prompts {
			tokenizer.Tokenize(p)
		}
		got := tokenizer.CollectTokens()
		want := test.Expected
		errMsg := CompareTokens(testId, got, want)
		if errMsg != "" {
			t.Errorf(errMsg)
		}
	}
	afterAll()
}

// If you want to add yor tests make sure you pass here trimmed single line input.
// Otherwise, add your tests to runner/runner_test.go file
func TestTokenizer(t *testing.T) {
	tests := []Test{
		{
			[]string{"ls = ls"},
			[]Token{
				{TokenType: WordToken, Value: "ls"},
				SpaceTokenInstance,
				AssignTokenInstance,
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "ls"},
			},
		},
		{
			[]string{"echo"},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
			},
		},
		{
			[]string{"echo some nice text"},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "some"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "nice"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "text"},
			},
		},
		{
			[]string{"echo check 'some nice text'Test"},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "check"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "some nice text"},
				{TokenType: WordToken, Value: "Test"},
			},
		},
		{
			[]string{"ls -l -a"},
			[]Token{
				{TokenType: WordToken, Value: "ls"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "-l"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "-a"},
			},
		},
		{
			[]string{"echo 'Hello World'"},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "Hello World"},
			},
		},
		{
			[]string{"echo \\\"Hello World\\\""},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "\\\"Hello"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "World\\\""},
			},
		},
		{
			[]string{"echo \\\"Hello \\\\\"World\\\\\"\\\""},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "\\\"Hello"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "\\\\"},
				{TokenType: StrongQuotationToken, Value: "World\\\\"},
				{TokenType: WordToken, Value: "\\\""},
			},
		},
		{
			[]string{"ls -l | grep myfile"},
			[]Token{
				{TokenType: WordToken, Value: "ls"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "-l"},
				PipeTokenInstance,
				{TokenType: WordToken, Value: "grep"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "myfile"},
			},
		},
		{
			[]string{"cat file1.txt file2.txt | grep 'keyword' | sed 's/old/new/g'"},
			[]Token{
				{TokenType: WordToken, Value: "cat"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "file1.txt"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "file2.txt"},
				PipeTokenInstance,
				{TokenType: WordToken, Value: "grep"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "keyword"},
				PipeTokenInstance,
				{TokenType: WordToken, Value: "sed"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "s/old/new/g"},
			},
		},
		{
			[]string{""},
			[]Token{},
		},
		{
			[]string{"echo 'She said \\\"Hello\\\"' | sed \"s/\\'/\\\"/g\""},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "She said \\\"Hello\\\""},
				PipeTokenInstance,
				{TokenType: WordToken, Value: "sed"},
				SpaceTokenInstance,
				{TokenType: StrongQuotationToken, Value: "s/\\'/\\\"/g"},
			},
		},
		{
			[]string{"echo 'abc\\\\' | sed 's/|/\\|/g' | grep \"\\\\\\\\\""},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "abc\\\\"},
				PipeTokenInstance,
				{TokenType: WordToken, Value: "sed"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "s/|/\\|/g"},
				PipeTokenInstance,
				{TokenType: WordToken, Value: "grep"},
				SpaceTokenInstance,
				{TokenType: StrongQuotationToken, Value: "\\\\\\\\"},
			},
		},
		{
			[]string{"echo 'a\\\\\\'b\\\\\\''' | grep \"\\\\\\\"c\\\"\""},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "a\\\\\\"},
				{TokenType: WordToken, Value: "b\\\\\\'"},
				PipeTokenInstance,
				{TokenType: WordToken, Value: "grep"},
				SpaceTokenInstance,
				{TokenType: StrongQuotationToken, Value: "\\\\\\\"c\\\""},
			},
		},
		{
			[]string{"echo 'a\\' | sed \"s/|/\\|/g\" | grep 'c\\|d\\|e'"},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "a\\"},
				PipeTokenInstance,
				{TokenType: WordToken, Value: "sed"},
				SpaceTokenInstance,
				{TokenType: StrongQuotationToken, Value: "s/|/\\|/g"},
				PipeTokenInstance,
				{TokenType: WordToken, Value: "grep"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "c\\|d\\|e"},
			},
		},
		{
			[]string{"echo 'abc\\\\\\' | sed \"s/|/\\|/g\" | grep 'c|d|e'"},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "abc\\\\\\"},
				PipeTokenInstance,
				{TokenType: WordToken, Value: "sed"},
				SpaceTokenInstance,
				{TokenType: StrongQuotationToken, Value: "s/|/\\|/g"},
				PipeTokenInstance,
				{TokenType: WordToken, Value: "grep"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "c|d|e"},
			},
		},
		{
			[]string{"echo \\'\\'"},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "\\'\\'"},
			},
		},
		{
			[]string{"echo 'Victor'\\''s'"},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WeakQuotationToken, Value: "Victor"},
				{TokenType: WordToken, Value: "\\'"},
				{TokenType: WeakQuotationToken, Value: "s"},
			},
		},
		{
			[]string{"echo Test\\value"},
			[]Token{
				{TokenType: WordToken, Value: "echo"},
				SpaceTokenInstance,
				{TokenType: WordToken, Value: "Test\\value"},
			},
		},
	}

	executeTokenizerTests(t, tests)
}

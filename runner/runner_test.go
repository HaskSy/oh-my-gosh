package runner

import (
	"fmt"
	. "gosh/config"
	. "gosh/tokenizer"
	"io"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	InitConfig()
	m.Run()
}

var runner Runner

func beforeEach(writer *io.PipeWriter, prompts []string) {
	runner.Clear()
	go func() {
		defer func(writer *io.PipeWriter) {
			err := writer.Close()
			if err != nil {
				panic(err)
			}
		}(writer)
		for _, prompt := range prompts {
			_, _ = writer.Write([]byte(fmt.Sprintf("%s\n", prompt)))
		}
	}()
}

func afterAll() {
	runner.Clear()
}

func executeMultilineTokenizerTests(t *testing.T, tests []Test) {
	runner = NewRunner()
	tmp, err := os.CreateTemp("", "example")
	if err != nil {
		panic(err)
	}

	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			panic(err)
		}
	}(tmp.Name())

	for testId, test := range tests {
		reader, writer := io.Pipe()
		beforeEach(writer, test.Prompts)
		err := runner.RunInteractive(reader, tmp, tmp, EmptyHandler)
		if err != nil {
			panic(err)
		}
		got := runner.tokenizer.CollectTokens()
		want := test.Expected
		errMsg := CompareTokens(testId, got, want)
		if errMsg != "" {
			t.Errorf(errMsg)
		}
	}
	afterAll()
}

// Each line should be written without <newline>,
// [executeMultilineTokenizerTests] will add <newline> itself
// Missing/Appearing Trailing SpaceTokenInstance is not really an issue for now.
// TODO Resolve Missing/Appearing Trailing SpaceTokenInstance issue
func TestRunner_MultilineTokenizer(t *testing.T) {
	tests := []Test{
		{
			[]string{},
			[]Token{},
		},
		{
			[]string{""},
			[]Token{},
		},
		{
			[]string{"echo"},
			[]Token{
				{WordToken, "echo"},
			},
		},
		{
			[]string{
				"multiline\\",
				"test",
			},
			[]Token{
				{WordToken, "multilinetest"},
				SpaceTokenInstance,
			},
		},
		{
			[]string{
				"git commit -m \"",
				"multiline",
				"commit",
				"\"",
			},
			[]Token{
				{WordToken, "git"},
				SpaceTokenInstance,
				{WordToken, "commit"},
				SpaceTokenInstance,
				{WordToken, "-m"},
				SpaceTokenInstance,
				{StrongQuotationToken, "multiline\ncommit\n"},
			},
		},
		{
			[]string{
				"git commit -m '",
				"multiline",
				"commit",
				"'",
			},
			[]Token{
				{WordToken, "git"},
				SpaceTokenInstance,
				{WordToken, "commit"},
				SpaceTokenInstance,
				{WordToken, "-m"},
				SpaceTokenInstance,
				{WeakQuotationToken, "multiline\ncommit\n"},
			},
		},
		{
			[]string{
				"echo \\",
				"Hello \\",
				"World",
			},
			[]Token{
				{WordToken, "echo"},
				SpaceTokenInstance,
				{WordToken, "Hello"},
				SpaceTokenInstance,
				{WordToken, "World"},
				SpaceTokenInstance,
			},
		},
		{
			[]string{
				"echo \\",
				"\"Hello\" \\",
				"World",
			},
			[]Token{
				{WordToken, "echo"},
				SpaceTokenInstance,
				{StrongQuotationToken, "Hello"},
				SpaceTokenInstance,
				{WordToken, "World"},
				SpaceTokenInstance,
			},
		},
		{
			[]string{
				"echo \\ |",
				"Hello \\ |",
				"World",
			},
			[]Token{
				{WordToken, "echo"},
				PipeTokenInstance,
				{WordToken, "Hello"},
				PipeTokenInstance,
				{WordToken, "World"},
				SpaceTokenInstance,
			},
		},
		{
			[]string{
				"echo |",
				"Hello |",
				"World",
			},
			[]Token{
				{WordToken, "echo"},
				PipeTokenInstance,
				{WordToken, "Hello"},
				PipeTokenInstance,
				{WordToken, "World"},
				SpaceTokenInstance,
			},
		},
		{
			[]string{
				"echo '",
				"Hello |",
				"World",
				"'",
			},
			[]Token{
				{WordToken, "echo"},
				SpaceTokenInstance,
				{WeakQuotationToken, "Hello |\nWorld\n"},
			},
		},
		{
			[]string{
				"echo \\ '",
				"Hello |",
				"World'",
			},
			[]Token{
				{WordToken, "echo"},
				SpaceTokenInstance,
				{WeakQuotationToken, "Hello |\nWorld"},
			},
		},
	}

	executeMultilineTokenizerTests(t, tests)
}

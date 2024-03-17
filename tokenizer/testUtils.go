package tokenizer

import (
	"fmt"
	"reflect"
)

type Test struct {
	Prompts  []string
	Expected []Token
}

func CompareTokens(testId int, got []Token, want []Token) string {
	if reflect.DeepEqual(got, want) {
		return ""
	}
	errMsg := ""
	minLength := len(got)
	if len(want) < minLength {
		minLength = len(want)
	}

	for i := 0; i < minLength; i++ {
		if got[i] != want[i] {
			errMsg += fmt.Sprintf("Difference in Test %d, at index %d: Expected %q, Got %q\n", testId, i, want[i], got[i])
		}
	}

	if len(got) > len(want) {
		for i := minLength; i < len(got); i++ {
			errMsg += fmt.Sprintf("Difference in Test %d, at index %d: Expected <empty>, Got %q\n", testId, i, got[i])
		}
	} else if len(got) < len(want) {
		for i := minLength; i < len(want); i++ {
			errMsg += fmt.Sprintf("Difference in Test %d, at index %d: Expected %q, Got <empty>\n", testId, i, want[i])
		}
	}
	return errMsg
}

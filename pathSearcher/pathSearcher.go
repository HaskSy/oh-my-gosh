package pathSearcher

import (
	shErr "gosh/shellError"
	"os"
	"strings"
)

type PathSearcher struct {
	paths []string
}

func (ps *PathSearcher) init() {
	ps.paths = strings.Split(os.Getenv("PATH"), ":")
}

func NewPathSearcher() PathSearcher {
	res := PathSearcher{}
	res.init()
	return res
}

func (ps *PathSearcher) FindBinary(name string) (string, shErr.ShellError) {

	if strings.HasPrefix(name, "/") {
		return name, nil
	}

	for _, path := range ps.paths {
		filePath := path + "/" + name
		if _, err := os.Stat(filePath); err == nil {
			return filePath, nil
		}
	}
	return "", shErr.NewCommand(name, "command not found")
}

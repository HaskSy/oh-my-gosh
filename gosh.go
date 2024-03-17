package main

import (
	. "gosh/config"
	. "gosh/runner"
	"os"
)

var mainRunner Runner

func init() {
	// Initialize the global configuration
	InitConfig()
	mainRunner = NewRunner()
}

func main() {
	err := mainRunner.RunInteractive(os.Stdin, os.Stderr, os.Stdout, DefaultHandler)
	if err != nil {
		panic(err)
	}
}

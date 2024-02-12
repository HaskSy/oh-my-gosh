package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("$ ")
		text, _, _ := reader.ReadLine()
		if bytes.Equal(text, []byte("exit")) {
			os.Exit(0)
		}
		fmt.Printf("%s\n", text)
	}
}

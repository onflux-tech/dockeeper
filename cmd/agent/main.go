package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("dockeeper-agent starting...")
	if len(os.Args) < 2 {
		fmt.Println("Usage: dockeeper-agent --manager-url=<url> --token=<token>")
		os.Exit(1)
	}
}

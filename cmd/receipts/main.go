package main

import (
	"os"

	"github.com/Ayush10212/receipts/adapters/cli"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "check" {
		args = args[1:]
	}
	os.Exit(cli.Run(args, os.Stdout, os.Stderr))
}

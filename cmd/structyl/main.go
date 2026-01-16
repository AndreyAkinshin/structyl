// Package main is the entry point for the structyl CLI.
package main

import (
	"os"

	"github.com/akinshin/structyl/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}

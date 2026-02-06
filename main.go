package main

import (
	"os"

	"github.com/iamgilwell/aura/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

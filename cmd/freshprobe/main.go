package main

import (
	"os"

	"github.com/Sudhan30/freshprobe/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

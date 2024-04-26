package main

import (
	"os"

	"github.com/thaddeusrhatcher/jirate/cmd"
)


func main() {
	rootCmd := cmd.NewRoot()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

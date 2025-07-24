package main

import (
	"os"

	"github.com/vzahanych/weather-demo-app/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

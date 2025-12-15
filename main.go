package main

import (
	"os"

	"cherry-go/cmd"
	"cherry-go/internal/logger"
)

func main() {
	logger.Init()

	if err := cmd.Execute(); err != nil {
		logger.Error("Failed to execute command: %v", err)
		os.Exit(1)
	}
}

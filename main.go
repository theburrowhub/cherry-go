package main

import (
	"cherry-go/cmd"
	"cherry-go/internal/logger"
	"os"
)

func main() {
	logger.Init()

	if err := cmd.Execute(); err != nil {
		logger.Error("Failed to execute command: %v", err)
		os.Exit(1)
	}
}

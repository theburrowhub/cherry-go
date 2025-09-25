package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ConfirmWithDefault asks for user confirmation with a default value
func ConfirmWithDefault(message string, defaultValue bool) bool {
	reader := bufio.NewReader(os.Stdin)
	
	// Prepare the prompt with default indication
	defaultText := "y/N"
	if defaultValue {
		defaultText = "Y/n"
	}
	
	fmt.Printf("%s [%s]: ", message, defaultText)
	
	input, err := reader.ReadString('\n')
	if err != nil {
		// If there's an error reading input, return default
		return defaultValue
	}
	
	// Clean the input
	input = strings.TrimSpace(strings.ToLower(input))
	
	// If empty input, use default
	if input == "" {
		return defaultValue
	}
	
	// Parse the response
	switch input {
	case "y", "yes", "true", "1":
		return true
	case "n", "no", "false", "0":
		return false
	default:
		// Invalid input, ask again
		fmt.Printf("Please answer yes or no.\n")
		return ConfirmWithDefault(message, defaultValue)
	}
}

// Confirm asks for user confirmation with default "yes"
func Confirm(message string) bool {
	return ConfirmWithDefault(message, true)
}

// IsInteractive checks if the current session is interactive
func IsInteractive() bool {
	// Check if stdin is a terminal
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	
	// If it's a character device (terminal), it's interactive
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ShouldPrompt determines if we should show prompts based on environment
func ShouldPrompt() bool {
	// Don't prompt in CI environments
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" || os.Getenv("GITLAB_CI") != "" {
		return false
	}
	
	// Don't prompt if explicitly disabled
	if os.Getenv("CHERRY_GO_NO_PROMPT") != "" {
		return false
	}
	
	// Only prompt if interactive
	return IsInteractive()
}

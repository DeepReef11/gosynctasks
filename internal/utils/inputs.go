package utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ReadInt reads an integer from stdin using bufio.NewReader
// This avoids the buffer issues that fmt.Scanf has with leftover newlines
func ReadInt() (int, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	value, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %w", err)
	}

	return value, nil
}

// ReadString reads a string from stdin using bufio.NewReader
// This avoids the buffer issues that fmt.Scanf has with leftover newlines
func ReadString() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return strings.TrimSpace(input), nil
}

// PromptYesNo prompts the user with a yes/no question and returns the result
func PromptYesNo(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s (y/n): ", question)
		response, _ := reader.ReadString('\n')
		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Please enter y or n")
			return PromptYesNo(question)
		}
	}
}

// PromptSelection displays a numbered list of items and prompts user to select one
// displayFunc is called for each item to display it
// Returns the selected index (0-based) and an error if cancelled or invalid input
func PromptSelection[T any](items []T, prompt string, displayFunc func(int, T)) (int, error) {
	if len(items) == 0 {
		return -1, fmt.Errorf("no items to select from")
	}

	// Display items
	for i, item := range items {
		displayFunc(i, item)
	}

	// Prompt for selection
	fmt.Printf("\n%s (1-%d) or 0 to cancel: ", prompt, len(items))
	choice, err := ReadInt()
	if err != nil {
		return -1, fmt.Errorf("invalid input: %w", err)
	}

	// Check for cancellation
	if choice == 0 {
		return -1, fmt.Errorf("operation cancelled")
	}

	// Validate choice
	if choice < 1 || choice > len(items) {
		return -1, fmt.Errorf("invalid choice: %d (must be 1-%d)", choice, len(items))
	}

	// Return 0-based index
	return choice - 1, nil
}

// PromptConfirmation displays a message and prompts for y/n confirmation
// Returns true if user confirms, false otherwise, and error for invalid input
func PromptConfirmation(message string) (bool, error) {
	fmt.Print(message + " (y/n): ")
	response, err := ReadString()
	if err != nil {
		return false, fmt.Errorf("invalid input: %w", err)
	}

	response = strings.ToLower(response)
	return response == "y" || response == "yes", nil
}

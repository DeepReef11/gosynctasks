package utils

import (
	"bufio"
	"fmt"
	// "log"
	"os"
	"strings"
)

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

// PromptChoice prompts the user to select a number from 0 to max (inclusive)
// Returns the selected number, or an error if the user cancelled (0) or provided invalid input
func PromptChoice(prompt string, min, max int) (int, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(prompt)
		response, err := reader.ReadString('\n')
		if err != nil {
			return 0, fmt.Errorf("failed to read input: %w", err)
		}

		response = strings.TrimSpace(response)
		var choice int
		_, err = fmt.Sscanf(response, "%d", &choice)
		if err != nil {
			fmt.Printf("Invalid input. Please enter a number between %d and %d.\n", min, max)
			continue
		}

		if choice == 0 {
			return 0, fmt.Errorf("operation cancelled")
		}

		if choice < min || choice > max {
			fmt.Printf("Invalid choice: %d. Please enter a number between %d and %d.\n", choice, min, max)
			continue
		}

		return choice, nil
	}
}

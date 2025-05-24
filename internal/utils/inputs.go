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

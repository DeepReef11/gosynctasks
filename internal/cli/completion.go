package cli

import (
	"gosynctasks/backend"
	"strings"

	"github.com/spf13/cobra"
)

// SmartCompletion provides shell completion for list names and actions
func SmartCompletion(taskLists []backend.TaskList) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var completions []string

		// First argument: suggest list names
		if len(args) == 0 {
			for _, list := range taskLists {
				if strings.HasPrefix(strings.ToLower(list.Name), strings.ToLower(toComplete)) {
					completions = append(completions, list.Name)
				}
			}
		}

		// Second argument (after list): suggest actions (full names only)
		if len(args) == 1 {
			actions := []string{"get", "add", "update", "complete"}
			for _, action := range actions {
				if strings.HasPrefix(action, strings.ToLower(toComplete)) {
					completions = append(completions, action)
				}
			}
		}

		// Third argument (after "<list> <action>"): no completion, user enters task summary
		// Return directive to stop completion
		if len(args) >= 2 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

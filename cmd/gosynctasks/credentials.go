package main

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"gosynctasks/internal/config"
	"gosynctasks/internal/credentials"
)

func newCredentialsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "credentials",
		Short: "Manage backend credentials",
		Long: `Securely manage credentials using system keyring.

Credentials can be stored in three ways (in priority order):
  1. System keyring (most secure) - recommended
  2. Environment variables (good for CI/CD)
  3. Config file URL (legacy - least secure)

Examples:
  # Store credentials in keyring (interactive password prompt)
  gosynctasks credentials set nextcloud myuser --prompt

  # Store credentials in keyring (non-interactive)
  gosynctasks credentials set nextcloud myuser mypassword

  # Check if credentials exist
  gosynctasks credentials get nextcloud myuser

  # Remove credentials from keyring
  gosynctasks credentials delete nextcloud myuser`,
	}

	cmd.AddCommand(newCredentialsSetCmd())
	cmd.AddCommand(newCredentialsGetCmd())
	cmd.AddCommand(newCredentialsDeleteCmd())

	return cmd
}

func newCredentialsSetCmd() *cobra.Command {
	var promptPassword bool

	cmd := &cobra.Command{
		Use:   "set <backend> [username] [password]",
		Short: "Store credentials in system keyring",
		Long: `Store backend credentials securely in the system keyring.

If username is not provided, it will be read from the backend configuration.
If --prompt is specified, password will be read interactively (recommended for security).

Examples:
  # Interactive password prompt (most secure)
  gosynctasks credentials set nextcloud myuser --prompt

  # Non-interactive (less secure - password visible in shell history)
  gosynctasks credentials set nextcloud myuser mypassword

  # Use username from config
  gosynctasks credentials set nextcloud --prompt`,
		Args: cobra.RangeArgs(1, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			backendName := args[0]

			// Get backend config to validate it exists
			cfg := config.GetConfig()
			backendConfig, exists := cfg.Backends[backendName]
			if !exists {
				return fmt.Errorf("backend %q not found in configuration", backendName)
			}

			// Determine username
			var username string
			if len(args) >= 2 {
				username = args[1]
			} else if backendConfig.Username != "" {
				username = backendConfig.Username
			} else {
				return fmt.Errorf("username is required (not found in config for backend %q)", backendName)
			}

			// Determine password
			var password string
			if promptPassword {
				// Interactive password prompt
				fmt.Printf("Enter password for %s@%s: ", username, backendName)
				passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println() // New line after password input
				if err != nil {
					return fmt.Errorf("failed to read password: %w", err)
				}
				password = string(passwordBytes)

				if password == "" {
					return fmt.Errorf("password cannot be empty")
				}
			} else if len(args) >= 3 {
				password = args[2]
			} else {
				return fmt.Errorf("password is required (use --prompt for interactive input)")
			}

			// Store in keyring
			if err := credentials.Set(backendName, username, password); err != nil {
				// Check if keyring is available
				if !credentials.IsAvailable() {
					return fmt.Errorf("system keyring is not available. Try using environment variables instead:\n  export GOSYNCTASKS_%s_USERNAME=%s\n  export GOSYNCTASKS_%s_PASSWORD=<password>",
						strings.ToUpper(strings.ReplaceAll(backendName, "-", "_")),
						username,
						strings.ToUpper(strings.ReplaceAll(backendName, "-", "_")))
				}
				return err
			}

			fmt.Printf("✓ Credentials stored successfully for %s@%s\n", username, backendName)
			fmt.Println("\nNext steps:")
			fmt.Printf("  1. Update your config to use keyring credentials:\n")
			if backendConfig.URL != "" {
				fmt.Printf("     - Remove password from URL\n")
			}
			if backendConfig.Host == "" {
				fmt.Printf("     - Add 'host: <hostname>' to backend config\n")
			}
			if backendConfig.Username == "" {
				fmt.Printf("     - Add 'username: %s' to backend config\n", username)
			}
			fmt.Printf("  2. Test the connection: gosynctasks %s\n", backendName)

			return nil
		},
	}

	cmd.Flags().BoolVar(&promptPassword, "prompt", false, "Prompt for password interactively (recommended)")

	return cmd
}

func newCredentialsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <backend> [username]",
		Short: "Check credential status for a backend",
		Long: `Check which credential source is being used for a backend.

This command shows where credentials are found (keyring, environment, or config URL)
but does not display the actual password for security reasons.

Examples:
  # Check credentials for backend
  gosynctasks credentials get nextcloud myuser

  # Use username from config
  gosynctasks credentials get nextcloud`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			backendName := args[0]

			// Get backend config
			cfg := config.GetConfig()
			backendConfig, exists := cfg.Backends[backendName]
			if !exists {
				return fmt.Errorf("backend %q not found in configuration", backendName)
			}

			// Determine username
			var username string
			if len(args) >= 2 {
				username = args[1]
			} else if backendConfig.Username != "" {
				username = backendConfig.Username
			}

			// Try to resolve credentials
			resolver := credentials.NewResolver()
			creds, err := resolver.ResolveWithConfig(backendName, username, backendConfig.Host, backendConfig.URL)

			if err != nil {
				fmt.Printf("✗ No credentials found for backend %q\n", backendName)
				fmt.Println("\nAvailable options:")
				fmt.Println("  1. Store in keyring:")
				fmt.Printf("     gosynctasks credentials set %s <username> --prompt\n", backendName)
				fmt.Println("  2. Set environment variables:")
				fmt.Printf("     export GOSYNCTASKS_%s_USERNAME=<username>\n", strings.ToUpper(strings.ReplaceAll(backendName, "-", "_")))
				fmt.Printf("     export GOSYNCTASKS_%s_PASSWORD=<password>\n", strings.ToUpper(strings.ReplaceAll(backendName, "-", "_")))
				fmt.Println("  3. Add to config URL (not recommended):")
				fmt.Printf("     url: \"nextcloud://username:password@host\"\n")
				return err
			}

			fmt.Printf("✓ Credentials found for backend %q\n", backendName)
			fmt.Printf("  Username: %s\n", creds.Username)
			fmt.Printf("  Source: %s\n", creds.Source)
			if creds.Host != "" {
				fmt.Printf("  Host: %s\n", creds.Host)
			}

			switch creds.Source {
			case credentials.SourceKeyring:
				fmt.Println("\n✓ Using secure keyring storage (recommended)")
			case credentials.SourceEnv:
				fmt.Println("\n⚠ Using environment variables")
				fmt.Println("  Consider using keyring for better security:")
				fmt.Printf("    gosynctasks credentials set %s %s --prompt\n", backendName, creds.Username)
			case credentials.SourceURL:
				fmt.Println("\n⚠ Using credentials from config URL (not recommended)")
				fmt.Println("  Consider migrating to keyring:")
				fmt.Printf("    gosynctasks credentials set %s %s --prompt\n", backendName, creds.Username)
				fmt.Println("  Then update config to remove credentials from URL")
			}

			return nil
		},
	}

	return cmd
}

func newCredentialsDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <backend> [username]",
		Short: "Remove credentials from system keyring",
		Long: `Remove stored credentials from the system keyring.

This only removes credentials from the keyring. Credentials in environment
variables or config file URL are not affected.

Examples:
  # Delete credentials (with confirmation)
  gosynctasks credentials delete nextcloud myuser

  # Delete credentials (skip confirmation)
  gosynctasks credentials delete nextcloud myuser --force

  # Use username from config
  gosynctasks credentials delete nextcloud`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			backendName := args[0]

			// Get backend config
			cfg := config.GetConfig()
			backendConfig, exists := cfg.Backends[backendName]
			if !exists {
				return fmt.Errorf("backend %q not found in configuration", backendName)
			}

			// Determine username
			var username string
			if len(args) >= 2 {
				username = args[1]
			} else if backendConfig.Username != "" {
				username = backendConfig.Username
			} else {
				return fmt.Errorf("username is required (not found in config for backend %q)", backendName)
			}

			// Confirm deletion unless --force
			if !force {
				fmt.Printf("Delete credentials for %s@%s from keyring? [y/N]: ", username, backendName)
				var response string
				n, err := fmt.Scanln(&response)
				if err != nil {
					fmt.Println("Error reading input:", err)
					return nil
				}
				if n == 0 {
					fmt.Println("No input was provided")
					return nil
				}
				response = strings.ToLower(strings.TrimSpace(response))
				if response != "y" && response != "yes" {
					fmt.Println("Cancelled")
					return nil
				}
			}

			// Delete from keyring
			if err := credentials.Delete(backendName, username); err != nil {
				return err
			}

			fmt.Printf("✓ Credentials removed for %s@%s\n", username, backendName)
			fmt.Println("\n⚠ Note: This only removed keyring credentials.")
			fmt.Println("  Environment variables and config URL credentials are not affected.")

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

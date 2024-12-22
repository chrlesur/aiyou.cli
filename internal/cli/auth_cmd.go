// Package cli provides the command-line interface implementation for the AI.YOU CLI
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/chrlesur/aiyou.cli/internal/api"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// newLoginCmd creates and returns the login command.
// This command handles user authentication with email and password.
func (a *App) newLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to AI.YOU API",
		Long: `Log in to AI.YOU API using your email and password.
   
   You can provide credentials through:
   - Command line flags (--email, --password)
   - Environment variables (AIYOU_EMAIL, AIYOU_PASSWORD)
   - .env file in the current directory
   - Interactive input (if no credentials are provided)`,
		Example: ` # Interactive login
	aiyou login
   
	# Login with flags
	aiyou login --email user@example.com --password mypassword
   
	# Login with environment variables
	export AIYOU_EMAIL=user@example.com
	export AIYOU_PASSWORD=mypassword
	aiyou login
   
	# Login with .env file
	# Create .env file with:
	# AIYOU_EMAIL=user@example.com
	# AIYOU_PASSWORD=mypassword
	aiyou login`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.IsLoggedIn() {
				a.logger.Warning("Login attempted while already logged in")
				return errors.New("already logged in, please logout first")
			}

			ctx := cmd.Context()
			var email, password string

			// Get email in order of precedence:
			// 1. Command line flags
			// 2. Environment variables (including .env)
			// 3. Interactive input
			email, _ = cmd.Flags().GetString("email")
			if email == "" {
				email = os.Getenv("AIYOU_EMAIL")
				if email == "" {
					a.logger.Debug("No email provided via flags or environment, requesting interactive input")
					fmt.Print("Email: ")
					fmt.Scanln(&email)
				} else {
					a.logger.Debug("Using email from environment")
				}
			} else {
				a.logger.Debug("Using email from command line flag")
			}

			// Basic email validation
			email = strings.TrimSpace(email)
			if !isValidEmail(email) {
				a.logger.Error("Invalid email format: %s", email)
				return fmt.Errorf("invalid email format: %s", email)
			}

			// Get password in order of precedence:
			// 1. Command line flags
			// 2. Environment variables (including .env)
			// 3. Interactive input
			password, _ = cmd.Flags().GetString("password")
			if password == "" {
				password = os.Getenv("AIYOU_PASSWORD")
				if password == "" {
					a.logger.Debug("No password provided via flags or environment, requesting interactive input")
					fmt.Print("Password: ")
					bytePassword, err := term.ReadPassword(int(syscall.Stdin))
					if err != nil {
						a.logger.Error("Failed to read password: %v", err)
						return fmt.Errorf("failed to read password: %w", err)
					}
					fmt.Println() // Add newline after password input
					password = string(bytePassword)
				} else {
					a.logger.Debug("Using password from environment")
				}
			} else {
				a.logger.Debug("Using password from command line flag")
			}

			// Attempt login
			a.logger.Debug("Attempting login for email: %s", email)
			if err := a.authManager.Login(ctx, email, password); err != nil {
				if errors.Is(err, api.ErrInvalidCredentials) {
					a.logger.Error("Login failed: invalid credentials")
					return errors.New("invalid email or password")
				}
				a.logger.Error("Login failed: %v", err)
				return fmt.Errorf("login failed: %w", err)
			}

			a.SetLoggedIn(true)
			a.logger.Info("Successfully logged in to AI.YOU")
			fmt.Println("Successfully logged in to AI.YOU")
			return nil
		},
	}

	// Add flags for command-line credential input
	cmd.Flags().String("email", "", "email address for login")
	cmd.Flags().String("password", "", "password for login")

	return cmd
}

// newLogoutCmd creates and returns the logout command.
// This command handles user logout and cleanup of authentication state.
func (a *App) newLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out from AI.YOU API",
		Long: `Log out from AI.YOU API and remove stored authentication credentials.
This will terminate your current session and require re-authentication for future commands.`,
		Example: ` # Logout
 aiyou logout`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !a.IsLoggedIn() {
				return errors.New("not logged in")
			}

			// Confirm logout
			fmt.Print("Are you sure you want to logout? (y/N): ")
			var confirm string
			fmt.Scanln(&confirm)
			if !strings.EqualFold(strings.TrimSpace(confirm), "y") {
				fmt.Println("Logout cancelled")
				return nil
			}

			// Attempt logout
			if err := a.authManager.Logout(); err != nil {
				return fmt.Errorf("logout failed: %w", err)
			}

			a.SetLoggedIn(false)
			fmt.Println("Successfully logged out from AI.YOU")
			return nil
		},
	}

	return cmd
}

// isValidEmail performs basic validation of email format.
func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) < 5 { // a@b.c minimum length
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return false
	}
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}

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

You can provide credentials either:
- Interactively (recommended)
- Via environment variables (AIYOU_EMAIL, AIYOU_PASSWORD)`,
		Example: ` # Interactive login (recommended)
 aiyou login

 # Login with environment variables
 export AIYOU_EMAIL=user@example.com
 export AIYOU_PASSWORD=mypassword
 aiyou login`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.IsLoggedIn() {
				return errors.New("already logged in, please logout first")
			}

			ctx := cmd.Context()
			var email, password string

			// Get email
			email = os.Getenv("AIYOU_EMAIL")
			if email == "" {
				fmt.Print("Email: ")
				fmt.Scanln(&email)
			}

			// Basic email validation
			email = strings.TrimSpace(email)
			if !isValidEmail(email) {
				return fmt.Errorf("invalid email format: %s", email)
			}

			// Get password
			password = os.Getenv("AIYOU_PASSWORD")
			if password == "" {
				fmt.Print("Password: ")
				bytePassword, err := term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return fmt.Errorf("failed to read password: %w", err)
				}
				fmt.Println() // Add newline after password input
				password = string(bytePassword)
			}

			// Attempt login
			if err := a.authManager.Login(ctx, email, password); err != nil {
				if errors.Is(err, api.ErrInvalidCredentials) {
					return errors.New("invalid email or password")
				}
				return fmt.Errorf("login failed: %w", err)
			}

			a.SetLoggedIn(true)
			fmt.Println("Successfully logged in to AI.YOU")
			return nil
		},
	}

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

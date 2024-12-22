// Package cli provides the command-line interface implementation
package cli

import (
	"fmt"
	"strings"

	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
	"github.com/spf13/cobra"
)

// newThreadCmd creates the parent command for thread management
func (a *App) newThreadCmd() *cobra.Command {
	a.logger.Debug("Creating thread command")

	threadCmd := &cobra.Command{
		Use:   "thread",
		Short: "Manage conversation threads",
		Long: `Create, list, view and delete conversation threads.

Examples:
 # List all threads
 aiyou thread list

 # View a specific thread
 aiyou thread view thread_123

 # Delete a thread
 aiyou thread delete thread_123

 # Create a new thread
 aiyou thread create`,
	}

	// Add subcommands
	threadCmd.AddCommand(
		a.newThreadListCmd(),
		a.newThreadViewCmd(),
		a.newThreadDeleteCmd(),
		a.newThreadCreateCmd(),
	)

	a.logger.Debug("Thread command created with subcommands")
	return threadCmd
}

// newThreadListCmd creates the command for listing threads
func (a *App) newThreadListCmd() *cobra.Command {
	a.logger.Debug("Creating thread list command")

	var page int
	var itemsPerPage int
	var search string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List conversation threads",
		Long: `List all conversation threads with pagination support.
 
You can specify the page number and items per page, and optionally filter threads with a search term.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			a.logger.Debug("Listing threads - Page: %d, ItemsPerPage: %d, Search: %s",
				page, itemsPerPage, search)

			params := &aiyou.UserThreadsParams{
				Page:         page,
				ItemsPerPage: itemsPerPage,
				Search:       search,
			}

			a.startProgress("Fetching threads")
			threads, err := a.threadManager.ListThreads(ctx, params)
			a.stopProgress()

			if err != nil {
				a.logger.Error("Failed to list threads: %v", err)
				return fmt.Errorf("failed to list threads: %w", err)
			}

			if len(threads.Threads) == 0 {
				a.logger.Info("No threads found")
				fmt.Println("No threads found.")
				return nil
			}

			// Display results in tabular format
			totalPages := (threads.TotalItems + threads.ItemsPerPage - 1) / threads.ItemsPerPage
			a.logger.Info("Found %d threads (Page %d of %d)",
				threads.TotalItems, threads.CurrentPage, totalPages)

			fmt.Printf("\nFound %d threads (Page %d of %d):\n\n",
				threads.TotalItems, threads.CurrentPage, totalPages)

			fmt.Printf("%-20s %-30s %-20s %s\n", "ID", "ASSISTANT", "CREATED", "FIRST MESSAGE")
			fmt.Println(strings.Repeat("-", 100))

			for _, thread := range threads.Threads {
				firstMsg := thread.FirstMessage
				if len(firstMsg) > 40 {
					firstMsg = firstMsg[:37] + "..."
				}

				fmt.Printf("%-20s %-30s %-20s %s\n",
					thread.ID,
					thread.AssistantName,
					thread.CreatedAt.Format("2006-01-02 15:04"),
					firstMsg)
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&itemsPerPage, "items", 10, "Items per page")
	cmd.Flags().StringVar(&search, "search", "", "Search term")

	a.logger.Debug("Thread list command created with flags")
	return cmd
}

// newThreadViewCmd creates the command for viewing a specific thread
func (a *App) newThreadViewCmd() *cobra.Command {
	a.logger.Debug("Creating thread view command")

	return &cobra.Command{
		Use:   "view [thread_id]",
		Short: "View a specific thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			threadID := args[0]

			a.logger.Debug("Viewing thread: %s", threadID)

			a.startProgress("Fetching thread")
			thread, err := a.threadManager.GetThread(ctx, threadID)
			a.stopProgress()

			if err != nil {
				a.logger.Error("Failed to get thread %s: %v", threadID, err)
				return fmt.Errorf("failed to get thread: %w", err)
			}

			a.logger.Info("Successfully retrieved thread %s", threadID)

			fmt.Printf("\nThread Details:\n")
			fmt.Printf("ID: %s\n", thread.ID)
			fmt.Printf("Assistant: %s\n", thread.AssistantName)
			fmt.Printf("Created: %s\n", thread.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("Updated: %s\n", thread.UpdatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("First Message: %s\n", thread.FirstMessage)

			return nil
		},
	}
}

// newThreadDeleteCmd creates the command for deleting a thread
func (a *App) newThreadDeleteCmd() *cobra.Command {
	a.logger.Debug("Creating thread delete command")

	var force bool
	cmd := &cobra.Command{
		Use:   "delete [thread_id]",
		Short: "Delete a conversation thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			threadID := args[0]

			a.logger.Debug("Attempting to delete thread %s (force: %v)", threadID, force)

			if !force {
				fmt.Printf("Are you sure you want to delete thread %s? [y/N]: ", threadID)
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(confirm) != "y" {
					a.logger.Info("Thread deletion cancelled by user")
					fmt.Println("Operation cancelled.")
					return nil
				}
			}

			a.startProgress("Deleting thread")
			err := a.threadManager.DeleteThread(ctx, threadID)
			a.stopProgress()

			if err != nil {
				a.logger.Error("Failed to delete thread %s: %v", threadID, err)
				return fmt.Errorf("failed to delete thread: %w", err)
			}

			a.logger.Info("Successfully deleted thread %s", threadID)
			fmt.Println("Thread deleted successfully.")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force deletion without confirmation")
	return cmd
}

// newThreadCreateCmd creates the command for creating a new thread
func (a *App) newThreadCreateCmd() *cobra.Command {
	a.logger.Debug("Creating thread create command")

	return &cobra.Command{
		Use:   "create",
		Short: "Create a new conversation thread",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			a.logger.Debug("Creating new thread")

			a.startProgress("Creating new thread")
			thread, err := a.threadManager.CreateThread(ctx)
			a.stopProgress()

			if err != nil {
				a.logger.Error("Failed to create thread: %v", err)
				return fmt.Errorf("failed to create thread: %w", err)
			}

			a.logger.Info("Successfully created new thread %s", thread.ID)

			fmt.Printf("\nNew thread created successfully:\n")
			fmt.Printf("ID: %s\n", thread.ID)
			fmt.Printf("Created: %s\n", thread.CreatedAt.Format("2006-01-02 15:04:05"))

			return nil
		},
	}
}

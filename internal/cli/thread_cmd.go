package cli

import (
	"fmt"
	"strings"

	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
	"github.com/spf13/cobra"
)

// newThreadCmd crée la commande parent pour la gestion des threads
func (a *App) newThreadCmd() *cobra.Command {
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

	// Ajouter les sous-commandes
	threadCmd.AddCommand(
		a.newThreadListCmd(),
		a.newThreadViewCmd(),
		a.newThreadDeleteCmd(),
		a.newThreadCreateCmd(),
	)

	return threadCmd
}

// newThreadListCmd crée la commande pour lister les threads
func (a *App) newThreadListCmd() *cobra.Command {
	var page int
	var itemsPerPage int
	var search string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List conversation threads",
		Long: `List all conversation threads with pagination support.
 
You can specify the page number and items per page, and optionally filter threads with a search term.`,
		Example: ` # List first page with default settings
 aiyou thread list

 # List specific page with custom page size
 aiyou thread list --page 2 --items 20

 # Search threads
 aiyou thread list --search "python"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Préparer les paramètres
			params := &aiyou.UserThreadsParams{
				Page:         page,
				ItemsPerPage: itemsPerPage,
				Search:       search,
			}

			// Récupérer les threads
			a.startProgress("Fetching threads")
			threads, err := a.threadManager.ListThreads(ctx, params)
			a.stopProgress()

			if err != nil {
				return fmt.Errorf("failed to list threads: %w", err)
			}

			// Afficher les résultats
			if len(threads.Threads) == 0 {
				fmt.Println("No threads found.")
				return nil
			}

			// Afficher les threads dans un format tabulaire
			fmt.Printf("\nFound %d threads (Page %d of %d):\n\n",
				threads.TotalItems,
				threads.CurrentPage,
				(threads.TotalItems+threads.ItemsPerPage-1)/threads.ItemsPerPage)

			fmt.Printf("%-20s %-30s %-20s %s\n", "ID", "ASSISTANT", "CREATED", "FIRST MESSAGE")
			fmt.Println(strings.Repeat("-", 100))

			for _, thread := range threads.Threads {
				// Tronquer le premier message s'il est trop long
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

	// Ajouter les flags
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&itemsPerPage, "items", 10, "Items per page")
	cmd.Flags().StringVar(&search, "search", "", "Search term")

	return cmd
}

// newThreadViewCmd crée la commande pour voir un thread spécifique
func (a *App) newThreadViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view [thread_id]",
		Short: "View a specific thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			threadID := args[0]

			a.startProgress("Fetching thread")
			thread, err := a.threadManager.GetThread(ctx, threadID)
			a.stopProgress()

			if err != nil {
				return fmt.Errorf("failed to get thread: %w", err)
			}

			// Afficher les détails du thread
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

// newThreadDeleteCmd crée la commande pour supprimer un thread
func (a *App) newThreadDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [thread_id]",
		Short: "Delete a conversation thread",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			threadID := args[0]

			if !force {
				// Demander confirmation
				fmt.Printf("Are you sure you want to delete thread %s? [y/N]: ", threadID)
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(confirm) != "y" {
					fmt.Println("Operation cancelled.")
					return nil
				}
			}

			a.startProgress("Deleting thread")
			err := a.threadManager.DeleteThread(ctx, threadID)
			a.stopProgress()

			if err != nil {
				return fmt.Errorf("failed to delete thread: %w", err)
			}

			fmt.Println("Thread deleted successfully.")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force deletion without confirmation")
	return cmd
}

// newThreadCreateCmd crée la commande pour créer un nouveau thread
func (a *App) newThreadCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a new conversation thread",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			a.startProgress("Creating new thread")
			thread, err := a.threadManager.CreateThread(ctx)
			a.stopProgress()

			if err != nil {
				return fmt.Errorf("failed to create thread: %w", err)
			}

			fmt.Printf("\nNew thread created successfully:\n")
			fmt.Printf("ID: %s\n", thread.ID)
			fmt.Printf("Created: %s\n", thread.CreatedAt.Format("2006-01-02 15:04:05"))

			return nil
		},
	}
}

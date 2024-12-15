// Package cli provides the command-line interface implementation.
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/chrlesur/aiyou.cli/internal/api"
	"github.com/chrlesur/aiyou.cli/internal/cache"
	"github.com/chrlesur/aiyou.cli/internal/cache/memory"
	"github.com/chrlesur/aiyou.cli/internal/config"
	"github.com/chrlesur/aiyou.cli/internal/interfaces"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// AppConfig holds the configuration for creating a new App instance
type AppConfig struct {
	Version string
	Config  *config.Config
	Logger  *logrus.Logger
}

// ClientAdapter adapte le client aiyou.Client pour implémenter l'interface AIClient
type ClientAdapter struct {
	*aiyou.Client
	isAuthenticated bool
	token           string
	mu              sync.RWMutex
}

// NewClientAdapter crée un nouvel adaptateur pour le client
func NewClientAdapter(client *aiyou.Client) *ClientAdapter {
	return &ClientAdapter{
		Client:          client,
		isAuthenticated: false,
	}
}

// GetToken implémente la méthode de l'interface
func (ca *ClientAdapter) GetToken() string {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return ca.token
}

// SetToken implémente la méthode de l'interface
func (ca *ClientAdapter) SetToken(token string) {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	ca.token = token
	ca.isAuthenticated = token != ""
}

// IsAuthenticated retourne l'état d'authentification
func (ca *ClientAdapter) IsAuthenticated() bool {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return ca.isAuthenticated
}

// Authenticate implémente la méthode de l'interface
func (ca *ClientAdapter) Authenticate(email, password string) error {
	if email == "" || password == "" {
		return fmt.Errorf("email and password are required")
	}
	ca.mu.Lock()
	ca.isAuthenticated = true
	ca.mu.Unlock()
	return nil
}

// RefreshToken implémente la méthode de l'interface
func (ca *ClientAdapter) RefreshToken() error {
	return nil
}

// App represents the CLI application structure
type App struct {
	rootCmd       *cobra.Command
	cfg           *config.Config
	log           *logrus.Logger
	version       string
	client        interfaces.AIClient
	cache         cache.Cache
	isLoggedIn    bool
	closeOnce     sync.Once
	isClosed      bool
	authManager   *api.AuthManager
	chatManager   *api.ChatManager
	threadManager *api.ThreadManager
	stdin         io.Reader
}

// NewApp creates and initializes a new CLI application
func NewApp(cfg *AppConfig) (*App, error) {
    if cfg == nil {
        return nil, fmt.Errorf("app config is required")
    }

    if cfg.Logger == nil {
        return nil, fmt.Errorf("logger is required")
    }

	// Initialize cache
	cacheConfig := cache.DefaultConfig()
	cacheInstance, err := memory.NewMemoryCache(cacheConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize aiyou.golib client
	baseClient, err := aiyou.NewClient("", "")
	if err != nil {
		cacheInstance.Close()
		return nil, fmt.Errorf("failed to initialize API client: %w", err)
	}

	// Create adapter
	clientAdapter := NewClientAdapter(baseClient)

	// Initialize auth manager
	authManager, err := api.NewAuthManager(baseClient, cfg.Config)
	if err != nil {
		cacheInstance.Close()
		return nil, fmt.Errorf("failed to initialize auth manager: %w", err)
	}

	// Initialize chat manager
	chatManager, err := api.NewChatManager(api.ChatManagerConfig{
		Client: clientAdapter,
		Cache:  cacheInstance,
		Config: cfg.Config,
		Logger: cfg.Logger,
	})
	if err != nil {
		cacheInstance.Close()
		return nil, fmt.Errorf("failed to initialize chat manager: %w", err)
	}
	// Initialize thread manager
	threadManager, err := api.NewThreadManager(api.ThreadManagerConfig{
		Client: clientAdapter,
		Cache:  cacheInstance,
		Config: cfg.Config,
		Logger: cfg.Logger,
	})
	if err != nil {
		cacheInstance.Close()
		return nil, fmt.Errorf("failed to initialize thread manager: %w", err)
	}

	app := &App{
		cfg:           cfg.Config,
		log:           cfg.Logger,
		version:       cfg.Version,
		client:        clientAdapter,
		cache:         cacheInstance,
		authManager:   authManager,
		chatManager:   chatManager,
		threadManager: threadManager,
		stdin:         os.Stdin,
	}

	app.initializeRootCommand()
	app.registerCommands()

	return app, nil
}

// Run executes the CLI application
func (a *App) Run(ctx context.Context) error {
	if a.isClosed {
		return fmt.Errorf("app is already closed")
	}

	defer func() {
		a.closeOnce.Do(func() {
			if a.cache != nil {
				if err := a.cache.Close(); err != nil {
					a.log.Errorf("Error closing cache: %v", err)
				}
			}
			a.isClosed = true
		})
	}()

	return a.rootCmd.ExecuteContext(ctx)
}

// Close properly closes the application resources
func (a *App) Close() error {
	var err error
	a.closeOnce.Do(func() {
		if a.cache != nil {
			err = a.cache.Close()
		}
		a.isClosed = true
	})
	return err
}

// initializeRootCommand sets up the root command and global flags
func (a *App) initializeRootCommand() {
	a.rootCmd = &cobra.Command{
		Use:   "aiyou",
		Short: "AI.YOU Command Line Interface",
		Long: `A powerful command line interface for interacting with AI.YOU services.
Complete documentation is available at https://docs.aiyou.cloud`,
		Version: a.version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip auth check for certain commands
			if cmd.Name() == "login" || cmd.Name() == "version" || cmd.Name() == "help" {
				return nil
			}

			// Verify authentication for other commands
			if !a.isLoggedIn {
				return fmt.Errorf("authentication required: please login first using 'aiyou login'")
			}

			return nil
		},
		SilenceUsage: true,
	}

	// Global flags
	a.rootCmd.PersistentFlags().Bool("debug", false, "enable debug mode")
	a.rootCmd.PersistentFlags().String("config", "", "config file (default is $HOME/.aiyou/config.yaml)")
	a.rootCmd.PersistentFlags().Bool("quiet", false, "suppress all non-error output")
}

// registerCommands adds all available commands to the CLI
func (a *App) registerCommands() {
	a.rootCmd.AddCommand(
		a.newLoginCmd(),
		a.newLogoutCmd(),
		a.newVersionCmd(),
		a.newCompletionCmd(),
		a.newConfigCmd(),
		a.newChatCmd(),
		a.newThreadCmd(),
	)
}

// newVersionCmd creates the version command
func (a *App) newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("aiyou CLI version %s\n", a.version)
		},
	}
}

// newCompletionCmd creates the completion command
func (a *App) newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate completion script",
		Long:      "Generate shell completion script for aiyou CLI",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				a.rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				a.rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				a.rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				a.rootCmd.GenPowerShellCompletion(os.Stdout)
			}
		},
	}
}

// newConfigCmd creates the config command
func (a *App) newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `View and modify configuration settings for aiyou CLI.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Configuration:\n")
			fmt.Printf(" API Endpoint: %s\n", a.cfg.APIEndpoint)
			fmt.Printf(" Log Level: %s\n", a.cfg.LogLevel)
			fmt.Printf(" Max Threads: %d\n", a.cfg.MaxThreads)
			fmt.Printf(" Debug Mode: %v\n", a.cfg.Debug)
		},
	}
}

// GetClient returns the API client
func (a *App) GetClient() interfaces.AIClient {
	return a.client
}

// SetLoggedIn sets the login status
func (a *App) SetLoggedIn(status bool) {
	a.isLoggedIn = status
}

// IsLoggedIn returns the current login status
func (a *App) IsLoggedIn() bool {
	return a.isLoggedIn
}

// startProgress displays a progress indicator with a message
func (a *App) startProgress(message string) {
	if !a.cfg.ShowProgress {
		return
	}
	fmt.Printf("%s... ", message)
}

// stopProgress stops the progress indicator
func (a *App) stopProgress() {
	if !a.cfg.ShowProgress {
		return
	}
	fmt.Println("done")
}


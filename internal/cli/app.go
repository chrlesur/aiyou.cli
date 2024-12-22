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
	"github.com/chrlesur/aiyou.cli/pkg/logger"
	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
	"github.com/spf13/cobra"
)

// AppConfig holds the configuration for creating a new App instance
type AppConfig struct {
	Version string
	Config  *config.Config
}

// ClientAdapter adapts the aiyou.Client to implement the AIClient interface
type ClientAdapter struct {
	*aiyou.Client
	isAuthenticated bool
	token           string
	mu              sync.RWMutex
	logger          *logger.Logger
}

// NewClientAdapter creates a new adapter for the client with proper logging
func NewClientAdapter(client *aiyou.Client) *ClientAdapter {
	log := logger.GetLogger()
	log.Debug("Creating new client adapter")

	return &ClientAdapter{
		Client:          client,
		logger:          log,
		isAuthenticated: false,
	}
}

// GetToken implements the interface method with logging
func (ca *ClientAdapter) GetToken() string {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	ca.logger.Debug("Getting token from adapter")
	return ca.token
}

// SetToken implements the interface method with logging
func (ca *ClientAdapter) SetToken(token string) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.logger.Debug("Setting new token")
	ca.token = token
	ca.isAuthenticated = token != ""
	ca.logger.Debug("Authentication status updated: %v", ca.isAuthenticated)
}

// IsAuthenticated returns the authentication status with logging
func (ca *ClientAdapter) IsAuthenticated() bool {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	ca.logger.Debug("Checking authentication status: %v", ca.isAuthenticated)
	return ca.isAuthenticated
}

// Authenticate implements the interface method with logging
func (ca *ClientAdapter) Authenticate(email, password string) error {
	ca.logger.Debug("Attempting authentication for email: %s", email)

	if email == "" || password == "" {
		ca.logger.Error("Authentication failed: empty credentials")
		return fmt.Errorf("email and password are required")
	}

	ca.mu.Lock()
	ca.isAuthenticated = true
	ca.mu.Unlock()

	ca.logger.Info("Authentication successful for email: %s", email)
	return nil
}

// RefreshToken implements the interface method with logging
func (ca *ClientAdapter) RefreshToken() error {
	ca.logger.Debug("Token refresh requested")
	return nil
}

// App represents the CLI application structure
type App struct {
	rootCmd       *cobra.Command
	cfg           *config.Config
	logger        *logger.Logger
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
	log := logger.GetLogger()
	log.Debug("Initializing new CLI application")

	if cfg == nil {
		log.Error("Failed to create app: config is required")
		return nil, fmt.Errorf("app config is required")
	}

	// Initialize cache
	log.Debug("Initializing cache")
	cacheConfig := cache.DefaultConfig()
	cacheInstance, err := memory.NewMemoryCache(cacheConfig)
	if err != nil {
		log.Error("Failed to initialize cache: %v", err)
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	// Initialize aiyou.golib client
	log.Debug("Initializing API client")
	baseClient, err := aiyou.NewClient("", "")
	if err != nil {
		cacheInstance.Close()
		log.Error("Failed to initialize API client: %v", err)
		return nil, fmt.Errorf("failed to initialize API client: %w", err)
	}

	// Create adapter
	clientAdapter := NewClientAdapter(baseClient)

	// Initialize auth manager
	log.Debug("Initializing auth manager")
	authManager, err := api.NewAuthManager(baseClient, cfg.Config)
	if err != nil {
		cacheInstance.Close()
		log.Error("Failed to initialize auth manager: %v", err)
		return nil, fmt.Errorf("failed to initialize auth manager: %w", err)
	}

	// Initialize chat manager
	log.Debug("Initializing chat manager")
	chatManager, err := api.NewChatManager(api.ChatManagerConfig{
		Client: clientAdapter,
		Cache:  cacheInstance,
		Config: cfg.Config,
	})
	if err != nil {
		cacheInstance.Close()
		log.Error("Failed to initialize chat manager: %v", err)
		return nil, fmt.Errorf("failed to initialize chat manager: %w", err)
	}

	// Initialize thread manager
	log.Debug("Initializing thread manager")
	threadManager, err := api.NewThreadManager(api.ThreadManagerConfig{
		Client: clientAdapter,
		Cache:  cacheInstance,
		Config: cfg.Config,
	})
	if err != nil {
		cacheInstance.Close()
		log.Error("Failed to initialize thread manager: %v", err)
		return nil, fmt.Errorf("failed to initialize thread manager: %w", err)
	}

	app := &App{
		cfg:           cfg.Config,
		logger:        log,
		version:       cfg.Version,
		client:        clientAdapter,
		cache:         cacheInstance,
		authManager:   authManager,
		chatManager:   chatManager,
		threadManager: threadManager,
		stdin:         os.Stdin,
	}

	log.Debug("Initializing root command")
	app.initializeRootCommand()
	app.registerCommands()

	log.Info("CLI application initialized successfully")
	return app, nil
}

// Run executes the CLI application
func (a *App) Run(ctx context.Context) error {
	a.logger.Debug("Starting CLI application")

	if a.isClosed {
		a.logger.Error("Cannot run: app is already closed")
		return fmt.Errorf("app is already closed")
	}

	defer func() {
		a.closeOnce.Do(func() {
			if a.cache != nil {
				if err := a.cache.Close(); err != nil {
					a.logger.Error("Error closing cache: %v", err)
				}
			}
			a.isClosed = true
			a.logger.Debug("Application resources cleaned up")
		})
	}()

	a.logger.Debug("Executing root command")
	return a.rootCmd.ExecuteContext(ctx)
}

// Close properly closes the application resources
func (a *App) Close() error {
	a.logger.Debug("Closing application")

	var err error
	a.closeOnce.Do(func() {
		if a.cache != nil {
			if closeErr := a.cache.Close(); closeErr != nil {
				a.logger.Error("Failed to close cache: %v", closeErr)
				err = closeErr
			}
		}
		a.isClosed = true
		a.logger.Info("Application closed successfully")
	})
	return err
}

// initializeRootCommand sets up the root command and global flags
func (a *App) initializeRootCommand() {
	a.logger.Debug("Setting up root command")

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
				a.logger.Warning("Command requires authentication but user is not logged in")
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

	a.logger.Debug("Root command initialized with flags")
}

// registerCommands adds all available commands to the CLI
func (a *App) registerCommands() {
	a.logger.Debug("Registering CLI commands")

	a.rootCmd.AddCommand(
		a.newLoginCmd(),
		a.newLogoutCmd(),
		a.newVersionCmd(),
		a.newCompletionCmd(),
		a.newConfigCmd(),
		a.newChatCmd(),
		a.newThreadCmd(),
	)

	a.logger.Debug("Commands registered successfully")
}

// GetClient returns the API client
func (a *App) GetClient() interfaces.AIClient {
	return a.client
}

// SetLoggedIn sets the login status
func (a *App) SetLoggedIn(status bool) {
	a.logger.Debug("Setting login status to: %v", status)
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
	a.logger.Debug("Starting progress: %s", message)
	fmt.Printf("%s... ", message)
}

// stopProgress stops the progress indicator
func (a *App) stopProgress() {
	if !a.cfg.ShowProgress {
		return
	}
	a.logger.Debug("Stopping progress indicator")
	fmt.Println("done")
}

// newVersionCmd creates the version command
func (a *App) newVersionCmd() *cobra.Command {
	a.logger.Debug("Creating version command")

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			a.logger.Debug("Executing version command")
			fmt.Printf("aiyou CLI version %s\n", a.version)
		},
	}

	return cmd
}

// newCompletionCmd creates the shell completion command
func (a *App) newCompletionCmd() *cobra.Command {
	a.logger.Debug("Creating completion command")

	cmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate completion script",
		Long:      "Generate shell completion script for aiyou CLI",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			a.logger.Debug("Generating completion script for shell: %s", args[0])

			var err error
			switch args[0] {
			case "bash":
				err = a.rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				err = a.rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				err = a.rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				err = a.rootCmd.GenPowerShellCompletion(os.Stdout)
			}

			if err != nil {
				a.logger.Error("Failed to generate completion script: %v", err)
			} else {
				a.logger.Info("Successfully generated completion script for %s", args[0])
			}
		},
	}

	return cmd
}

// newConfigCmd creates the config command
func (a *App) newConfigCmd() *cobra.Command {
	a.logger.Debug("Creating config command")

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `View and modify configuration settings for aiyou CLI.`,
		Run: func(cmd *cobra.Command, args []string) {
			a.logger.Debug("Displaying current configuration")

			fmt.Printf("Configuration:\n")
			fmt.Printf(" API Endpoint: %s\n", a.cfg.APIEndpoint)
			fmt.Printf(" Log Level: %s\n", a.cfg.LogLevel)
			fmt.Printf(" Max Threads: %d\n", a.cfg.MaxThreads)
			fmt.Printf(" Debug Mode: %v\n", a.cfg.Debug)

			a.logger.Debug("Configuration displayed successfully")
		},
	}

	return cmd
}

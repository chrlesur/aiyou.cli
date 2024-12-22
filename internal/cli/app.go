// Package cli provides the command-line interface implementation.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

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

	// Initialize API client
	log.Debug("Initializing API client")
	baseClient, err := aiyou.NewClient("", "")
	if err != nil {
		cacheInstance.Close()
		log.Error("Failed to initialize API client: %v", err)
		return nil, fmt.Errorf("failed to initialize API client: %w", err)
	}

	clientAdapter := NewClientAdapter(baseClient)

	// Initialize auth manager
	authManager, err := api.NewAuthManager(baseClient, cfg.Config)
	if err != nil {
		cacheInstance.Close()
		log.Error("Failed to initialize auth manager: %v", err)
		return nil, fmt.Errorf("failed to initialize auth manager: %w", err)
	}

	// Vérifier l'authentification existante
	ctx := context.Background()
	token, err := authManager.GetToken(ctx)
	isLoggedIn := err == nil && token != ""
	if isLoggedIn {
		log.Debug("Found existing token, applying to client")
		clientAdapter.SetToken(token)
	} else {
		log.Debug("No existing token found")
	}
	log.Debug("Initial authentication status: %v", isLoggedIn)

	// Initialize chat manager
	chatManager, err := api.NewChatManager(api.ChatManagerConfig{
		Client: clientAdapter, // Utiliser le clientAdapter avec le token
		Cache:  cacheInstance,
		Config: cfg.Config,
	})
	if err != nil {
		cacheInstance.Close()
		log.Error("Failed to initialize chat manager: %v", err)
		return nil, fmt.Errorf("failed to initialize chat manager: %w", err)
	}

	// Initialize thread manager
	threadManager, err := api.NewThreadManager(api.ThreadManagerConfig{
		Client: clientAdapter, // Utiliser le clientAdapter avec le token
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
		isLoggedIn:    isLoggedIn,
	}

	app.initializeRootCommand()
	app.registerCommands()

	log.Info("CLI application initialized successfully")
	return app, nil
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
			verbose, _ := cmd.Flags().GetBool("verbose")
			debug, _ := cmd.Flags().GetBool("debug")

			// Configure log level based on flags
			if debug {
				a.logger.SetLevel(logger.DebugLevel)
				a.logger.Debug("Debug mode enabled")
			} else if verbose {
				a.logger.SetLevel(logger.InfoLevel)
				a.logger.Info("Verbose mode enabled")
			} else {
				a.logger.SetLevel(logger.WarningLevel)
			}

			return nil
		},
		SilenceUsage: true,
	}

	// Global flags
	a.rootCmd.PersistentFlags().Bool("debug", false, "enable debug mode (detailed debug information)")
	a.rootCmd.PersistentFlags().Bool("verbose", false, "enable verbose mode (informational output)")
	a.rootCmd.PersistentFlags().String("config", "", "config file (default is $HOME/.aiyou/config.yaml)")
}

func (a *App) newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			debug, _ := cmd.Flags().GetBool("debug")

			if debug {
				// Mode debug : informations très détaillées
				a.logger.Debug("Command: version")
				a.logger.Debug("Flags:")
				a.logger.Debug(" - verbose: %v", verbose)
				a.logger.Debug(" - debug: %v", debug)
				a.logger.Debug("Build info:")
				a.logger.Debug(" - Version: %s", a.version)
				a.logger.Debug(" - Go version: %s", runtime.Version())
				a.logger.Debug(" - OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
				fmt.Printf("aiyou CLI version %s (%s/%s)\n",
					a.version, runtime.GOOS, runtime.GOARCH)
			} else if verbose {
				// Mode verbose : informations basiques supplémentaires
				a.logger.Info("CLI Version: %s", a.version)
				a.logger.Info("OS: %s", runtime.GOOS)
				fmt.Printf("aiyou CLI version %s (%s)\n",
					a.version, runtime.GOOS)
			} else {
				// Mode normal : juste la version
				fmt.Printf("aiyou version %s\n", a.version)
			}
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return a.rootCmd.PersistentPreRunE(cmd, args)
		},
	}
}

// Run executes the CLI application
func (a *App) Run(ctx context.Context) error {
	a.logger.Debug("Starting CLI application")

	if a.isClosed {
		a.logger.Error("Cannot run: app is already closed")
		return fmt.Errorf("app is already closed")
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		a.logger.Info("Received shutdown signal")
		a.Close()
	}()

	return a.rootCmd.ExecuteContext(ctx)
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
}

// Close properly closes the application resources
func (a *App) Close() error {
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
	return &ClientAdapter{
		Client:          client,
		isAuthenticated: false,
		logger:          logger.GetLogger(),
	}
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

// newCompletionCmd creates the completion command
func (a *App) newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate completion script",
		Long:      "Generate shell completion script for aiyou CLI",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.ExactValidArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			switch args[0] {
			case "bash":
				err = a.rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				err = a.rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				err = a.rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				err = a.rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			if err != nil {
				a.logger.Error("Failed to generate completion script: %v", err)
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
			a.logger.Debug("Displaying current configuration")
			fmt.Printf("Configuration:\n")
			fmt.Printf(" API Endpoint: %s\n", a.cfg.APIEndpoint)
			fmt.Printf(" Log Level: %s\n", a.cfg.LogLevel)
			fmt.Printf(" Max Threads: %d\n", a.cfg.MaxThreads)
			fmt.Printf(" Debug Mode: %v\n", a.cfg.Debug)
		},
	}
}

// ClientAdapter implementation of interfaces.AIClient
func (ca *ClientAdapter) Authenticate(email, password string) error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.logger.Debug("Attempting authentication for email: %s", email)

	newClient, err := aiyou.NewClient(email, password)
	if err != nil {
		ca.logger.Error("Authentication failed: %v", err)
		return err
	}

	ca.Client = newClient
	ca.isAuthenticated = true
	ca.logger.Info("Authentication successful for email: %s", email)
	return nil
}

func (ca *ClientAdapter) GetToken() string {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return ca.token
}

func (ca *ClientAdapter) IsAuthenticated() bool {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	return ca.isAuthenticated && ca.token != ""
}

func (ca *ClientAdapter) SetToken(token string) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.logger.Debug("Setting token in client adapter")
	ca.token = token

	// Si on a un token, créer un nouveau client authentifié
	if token != "" {
		// Utiliser le même client mais avec le token
		ca.isAuthenticated = true
	} else {
		ca.isAuthenticated = false
	}

	ca.logger.Debug("Client authentication status updated: %v", ca.isAuthenticated)
}

func (ca *ClientAdapter) RefreshToken() error {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	ca.logger.Debug("Refreshing token")
	// Implement token refresh logic here if needed
	return nil
}

func (ca *ClientAdapter) CreateChatCompletion(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.ChatCompletionResponse, error) {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	if !ca.isAuthenticated || ca.Client == nil {
		return nil, errors.New("not authenticated")
	}

	return ca.Client.CreateChatCompletion(ctx, messages, assistantID)
}

func (ca *ClientAdapter) CreateChatCompletionStream(ctx context.Context, messages []aiyou.Message, assistantID string) (*aiyou.StreamReader, error) {
	ca.logger.Debug("Creating chat completion stream with assistant: %s", assistantID)
	return ca.Client.CreateChatCompletionStream(ctx, messages, assistantID)
}

func (ca *ClientAdapter) SaveConversation(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error) {
	ca.logger.Debug("Saving conversation")
	return ca.Client.SaveConversation(ctx, req)
}

func (ca *ClientAdapter) GetConversation(ctx context.Context, threadID string) (*aiyou.ConversationThread, error) {
	ca.logger.Debug("Getting conversation thread: %s", threadID)
	return ca.Client.GetConversation(ctx, threadID)
}

func (ca *ClientAdapter) GetUserThreads(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error) {
	ca.logger.Debug("Getting user threads")
	return ca.Client.GetUserThreads(ctx, params)
}

func (ca *ClientAdapter) DeleteThread(ctx context.Context, threadID string) error {
	ca.logger.Debug("Deleting thread: %s", threadID)
	return ca.Client.DeleteThread(ctx, threadID)
}

func (ca *ClientAdapter) GetUserAssistants(ctx context.Context) (*aiyou.AssistantsResponse, error) {
	ca.logger.Debug("Getting user assistants")
	return ca.Client.GetUserAssistants(ctx)
}

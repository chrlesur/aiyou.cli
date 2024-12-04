// internal/cli/thread_cmd_test.go

package cli

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chrlesur/aiyou.golib/pkg/aiyou"
	"github.com/stretchr/testify/assert"
)

func TestThreadCommands(t *testing.T) {
	app, mockClient, cleanup := setupAuthTest(t)
	defer cleanup()

	// Configuration initiale
	app.SetLoggedIn(true)
	mockClient.IsAuthenticatedFn = func() bool {
		return true
	}
	app.chatManager.SetTestClient(mockClient)
	app.threadManager.SetTestClient(mockClient)

	// Initialisation du nombre maximum de threads
	maxThreads := 5
	app.cfg.MaxThreads = maxThreads

	t.Run("thread list command", func(t *testing.T) {
		mockClient.GetUserThreadsFn = func(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error) {
			return &aiyou.UserThreadsOutput{
				Threads:      []aiyou.ConversationThread{},
				TotalItems:   0,
				ItemsPerPage: 10,
				CurrentPage:  1,
			}, nil
		}

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		app.rootCmd.SetArgs([]string{"thread", "list"})
		err := app.rootCmd.Execute()
		assert.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		var output strings.Builder
		io.Copy(&output, r)

		assert.Contains(t, output.String(), "No threads found")
	})

	t.Run("thread create command", func(t *testing.T) {
		mockClient.GetUserThreadsFn = func(ctx context.Context, params *aiyou.UserThreadsParams) (*aiyou.UserThreadsOutput, error) {
			return &aiyou.UserThreadsOutput{
				Threads:    make([]aiyou.ConversationThread, 0), // Liste vide de threads
				TotalItems: 0,
			}, nil
		}
		mockClient.SaveConversationFn = func(ctx context.Context, req aiyou.SaveConversationRequest) (*aiyou.SaveConversationResponse, error) {
			return &aiyou.SaveConversationResponse{ID: "new_thread_1"}, nil
		}
		mockClient.GetConversationFn = func(ctx context.Context, threadID string) (*aiyou.ConversationThread, error) {
			return &aiyou.ConversationThread{
				ID:        threadID,
				CreatedAt: time.Now(),
			}, nil
		}

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		app.rootCmd.SetArgs([]string{"thread", "create"})
		err := app.rootCmd.Execute()
		assert.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		var output strings.Builder
		io.Copy(&output, r)

		assert.Contains(t, output.String(), "New thread created successfully")
		assert.Contains(t, output.String(), "ID: new_thread_1")
	})
}

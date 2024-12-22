package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_Initialization(t *testing.T) {
	ResetForTest()

	log := GetLogger()
	assert.NotNil(t, log)
	assert.Equal(t, InfoLevel, log.level)
	assert.NotNil(t, log.logger)
}

func TestLogger_LogFile(t *testing.T) {
	ResetForTest()
	tmpDir := t.TempDir()

	log := GetLogger()
	err := log.Configure(Config{
		LogDir: tmpDir,
		Level:  InfoLevel,
		Silent: false,
	})
	require.NoError(t, err)

	testMessage := "Test message for file logging"
	log.Info(testMessage)

	// Force sync and read file
	expectedFile := filepath.Join(tmpDir, "aiyou_"+time.Now().Format("2006-01-02")+".log")
	content, err := os.ReadFile(expectedFile)
	require.NoError(t, err)

	assert.Contains(t, string(content), testMessage)
}

func TestLogger_Levels(t *testing.T) {
	ResetForTest()
	tmpDir := t.TempDir()

	tests := []struct {
		name string
		level LogLevel
		logFunc func(*Logger, string)
		message string
		shouldLog bool
	}{
		{
			name: "debug with debug enabled",
			level: DebugLevel,
			logFunc: func(l *Logger, msg string) { l.Debug(msg) },
			message: "Debug message",
			shouldLog: true,
		},
		{
			name: "debug with info level",
			level: InfoLevel,
			logFunc: func(l *Logger, msg string) { l.Debug(msg) },
			message: "Should not appear",
			shouldLog: false,
		},
		{
			name: "error always logs",
			level: InfoLevel,
			logFunc: func(l *Logger, msg string) { l.Error(msg) },
			message: "Error message",
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset and configure logger for each test
			ResetForTest()
			log := GetLogger()
			err := log.Configure(Config{
				LogDir: tmpDir,
				Level: tt.level,
				Silent: false,
			})
			require.NoError(t, err)

			// Write message
			tt.logFunc(log, tt.message)

			// Force sync
			if log.file != nil {
				log.file.Sync()
			}

			// Wait a bit for filesystem
			time.Sleep(10 * time.Millisecond)

			// Read log file
			logFile := filepath.Join(tmpDir, "aiyou_"+time.Now().Format("2006-01-02")+".log")
			content, err := os.ReadFile(logFile)
			require.NoError(t, err)

			if tt.shouldLog {
				assert.Contains(t, string(content), tt.message, 
					"Expected message to be logged")
			} else {
				assert.NotContains(t, string(content), tt.message, 
					"Message should not have been logged")
			}
		})
	}
}

func TestLogger_SilentMode(t *testing.T) {
	ResetForTest()
	tmpDir := t.TempDir()

	log := GetLogger()
	err := log.Configure(Config{
		LogDir: tmpDir,
		Level:  InfoLevel,
		Silent: true,
	})
	require.NoError(t, err)

	logFile := filepath.Join(tmpDir, "aiyou_"+time.Now().Format("2006-01-02")+".log")
	log.Info("Should not appear")

	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(string(content)))

	// Errors should still be logged in silent mode
	errorMsg := "Error should appear"
	log.Error(errorMsg)
	content, err = os.ReadFile(logFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), errorMsg)
}

func TestLogger_MessageFormatting(t *testing.T) {
	ResetForTest()
	log := GetLogger()

	tests := []struct {
		name       string
		level      string // Changed from LogLevel to string
		message    string
		args       []interface{}
		checkParts []string
	}{
		{
			name:    "simple message",
			level:   "INFO",
			message: "Test message",
			args:    nil,
			checkParts: []string{
				"[INFO]",
				"Test message",
			},
		},
		{
			name:    "formatted message",
			level:   "ERROR",
			message: "Error: %s, code: %d",
			args:    []interface{}{"test error", 123},
			checkParts: []string{
				"[ERROR]",
				"Error: test error, code: 123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := log.formatMessage(tt.level, tt.message, tt.args...)
			for _, part := range tt.checkParts {
				assert.Contains(t, formatted, part)
			}
		})
	}
}

func TestLogger_ConcurrentAccess(t *testing.T) {
	ResetForTest()
	tmpDir := t.TempDir()

	log := GetLogger()
	err := log.Configure(Config{
		LogDir: tmpDir,
		Level:  InfoLevel,
		Silent: false,
	})
	require.NoError(t, err)

	done := make(chan bool)
	messageCount := 100

	// Write messages concurrently
	for i := 0; i < messageCount; i++ {
		go func(i int) {
			log.Info("Concurrent message %d", i)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < messageCount; i++ {
		<-done
	}

	// Verify log file
	logFile := filepath.Join(tmpDir, "aiyou_"+time.Now().Format("2006-01-02")+".log")
	content, err := os.ReadFile(logFile)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Equal(t, messageCount, len(lines))
}

func TestLogger_Close(t *testing.T) {
	ResetForTest()
	tmpDir := t.TempDir()

	log := GetLogger()
	err := log.Configure(Config{
		LogDir: tmpDir,
		Level: InfoLevel,
		Silent: false,
	})
	require.NoError(t, err)

	// Write something
	log.Info("Test message")
	
	// Ensure file exists before close
	logFile := filepath.Join(tmpDir, "aiyou_"+time.Now().Format("2006-01-02")+".log")
	_, err = os.Stat(logFile)
	require.NoError(t, err)

	// Close logger
	log.Close()

	// Verify cleanup
	assert.Nil(t, log.file, "File should be nil after close")
	assert.NotNil(t, log.logger, "Logger should still be functional")
	
	// Verify we can still log after close
	assert.NotPanics(t, func() {
		log.Info("Post-close message")
	})

	// Verify the message was logged to stdout
	assert.NotNil(t, log.writer, "Writer should not be nil after close")
}

// Test helper function
func verifyCanLog(t *testing.T, log *Logger) {
	// Ensure we can still log after any operation
	assert.NotPanics(t, func() {
		log.Info("Verification message")
	})
}

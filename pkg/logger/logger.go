// Package logger provides a thread-safe logging system with support for
// multiple output destinations and log levels.
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	// DebugLevel is used for detailed system operations
	DebugLevel LogLevel = iota
	// InfoLevel is used for general operational entries
	InfoLevel
	// WarningLevel is used for non-critical issues
	WarningLevel
	// ErrorLevel is used for errors that need attention
	ErrorLevel
)

// Config holds logger configuration options
type Config struct {
	LogDir string   // Directory where log files will be stored
	Level  LogLevel // Minimum level of messages to log
	Silent bool     // If true, suppresses all output except errors
}

// Global logger instance with thread-safe initialization
var (
	instance *Logger
	once     sync.Once
)

// Logger handles all logging operations with support for
// different log levels, file output, and concurrent access.
type Logger struct {
	level  LogLevel
	logger *log.Logger
	file   *os.File
	mu     sync.Mutex
	silent bool
	logDir string
	writer io.Writer
}

// GetLogger returns the singleton instance of Logger.
// It ensures thread-safe initialization with default configuration.
func GetLogger() *Logger {
	once.Do(func() {
		instance = &Logger{
			level:  InfoLevel,
			logDir: "logs",
			silent: false,
		}
		// Initialize with default stdout logging
		instance.writer = os.Stdout
		instance.logger = log.New(os.Stdout, "", log.LstdFlags)
	})
	return instance
}

// Configure initializes or reconfigures the logger with the provided configuration.
// It manages file handles and writers appropriately.
func (l *Logger) Configure(cfg Config) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Cleanup existing resources
	if l.file != nil {
		l.file.Close()
	}

	l.level = cfg.Level
	l.silent = cfg.Silent
	l.logDir = cfg.LogDir

	// Configure file output if directory is specified
	if l.logDir != "" {
		if err := os.MkdirAll(l.logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		logFile := filepath.Join(l.logDir, fmt.Sprintf("aiyou_%s.log", time.Now().Format("2006-01-02")))
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		l.file = file
		l.writer = io.MultiWriter(os.Stdout, file)
	} else {
		l.writer = os.Stdout
	}

	l.logger = log.New(l.writer, "", log.LstdFlags)
	return nil
}

// SetLevel sets the minimum log level that will be output
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetSilentMode enables or disables silent mode
func (l *Logger) SetSilentMode(silent bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.silent = silent
}

// writeLog handles the actual writing of log messages with proper synchronization
func (l *Logger) writeLog(level, message string, args ...interface{}) {
	if l == nil || l.logger == nil {
		fmt.Printf("[%s] %s\n", level, fmt.Sprintf(message, args...))
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	msg := l.formatMessage(level, message, args...)
	_, err := fmt.Fprintln(l.writer, msg)
	if err != nil {
		fmt.Printf("Error writing to log: %v\n", err)
	}

	if l.file != nil {
		l.file.Sync()
	}
}

// formatMessage formats a log message with timestamp and level
func (l *Logger) formatMessage(level string, message string, args ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formattedMsg := fmt.Sprintf(message, args...)

	if l.level <= DebugLevel {
		_, file, line, _ := runtime.Caller(2)
		return fmt.Sprintf("[%s] %s:%d - [%s] %s",
			timestamp,
			filepath.Base(file),
			line,
			level,
			formattedMsg)
	}

	return fmt.Sprintf("[%s] [%s] %s", timestamp, level, formattedMsg)
}

// Debug logs a message at DebugLevel
func (l *Logger) Debug(message string, args ...interface{}) {
	if l.level <= DebugLevel && !l.silent {
		l.writeLog("DEBUG", message, args...)
	}
}

// Info logs a message at InfoLevel
func (l *Logger) Info(message string, args ...interface{}) {
	if l.level <= InfoLevel && !l.silent {
		l.writeLog("INFO", message, args...)
	}
}

// Warning logs a message at WarningLevel
func (l *Logger) Warning(message string, args ...interface{}) {
	if l.level <= WarningLevel && !l.silent {
		l.writeLog("WARNING", message, args...)
	}
}

// Error logs a message at ErrorLevel
func (l *Logger) Error(message string, args ...interface{}) {
	// Errors are always logged, regardless of silent mode
	l.writeLog("ERROR", message, args...)
}

// Close properly closes the log file
// Dans logger.go
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.Sync()
		l.file.Close()
		l.file = nil
	}

	// Reset to stdout only
	l.writer = os.Stdout
	l.logger = log.New(os.Stdout, "", log.LstdFlags)
}

// ResetForTest resets the logger state for testing purposes
func ResetForTest() {
	once = sync.Once{}
	if instance != nil && instance.file != nil {
		instance.file.Close()
	}
	instance = nil
}

// String returns the string representation of a LogLevel
func (l LogLevel) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarningLevel:
		return "WARNING"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

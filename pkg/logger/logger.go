package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	debugMode  bool
	silentMode bool
	logger     *log.Logger
)

func init() {
	logger = log.New(os.Stdout, "", 0)
}

func SetDebugMode(debug bool) {
	debugMode = debug
}

func SetSilentMode(silent bool) {
	silentMode = silent
}

func formatMessage(level, message string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	return fmt.Sprintf("[%s] %s: %s", timestamp, level, message)
}

func Info(message string) {
	if !silentMode {
		logger.Println(formatMessage("INFO", message))
	}
}

func Debug(message string) {
	if debugMode && !silentMode {
		logger.Println(formatMessage("DEBUG", message))
	}
}

func Error(message string) {
	// Errors are always logged, even in silent mode
	logger.Println(formatMessage("ERROR", message))
}

func Warning(message string) {
	if !silentMode {
		logger.Println(formatMessage("WARNING", message))
	}
}

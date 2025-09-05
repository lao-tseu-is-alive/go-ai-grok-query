package config

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

// GetLogWriterFromEnvOrPanic returns the name of the filename to use for LOG from the content of the env variable :
// LOG_FILE : string containing the filename to use for LOG, use DISCARD for no log, default is STDERR
func GetLogWriterFromEnvOrPanic(defaultLogName string) io.Writer {
	logFileName := defaultLogName
	val, exist := os.LookupEnv("LOG_FILE")
	if exist {
		logFileName = val
	}
	if utf8.RuneCountInString(logFileName) < 5 {
		panic(fmt.Sprintf("💥💥 error env LOG_FILE filename should contain at least %d characters (got %d).",
			5, utf8.RuneCountInString(val)))
	}
	switch logFileName {
	case "stdout":
		return os.Stdout
	case "stderr":
		return os.Stderr
	case "DISCARD":
		return io.Discard
	default:
		// Open the file with append, create, and write permissions.
		// The 0644 permission allows the owner to read/write and others to read.
		file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// Return an error if the file cannot be opened (e.g., due to permissions).
			panic(fmt.Sprintf("💥💥 ERROR: LOG_FILE %q could not be open : %v", logFileName, err))
		}
		return file
	}
}

// GetLogLevelFromEnvOrPanic reads LOG_LEVEL from environment and returns a golog.Level.
// Accepted values (case-insensitive): debug, info, warn, error, fatal, or their numeric equivalents (0-4).
// If unset, returns defaultLevel. Panics on invalid value.
func GetLogLevelFromEnvOrPanic(defaultLevel golog.Level) golog.Level {
	val, ok := os.LookupEnv("LOG_LEVEL")
	if !ok || strings.TrimSpace(val) == "" {
		return defaultLevel
	}

	v := strings.TrimSpace(strings.ToLower(val))
	switch v {
	case "debug", "0":
		return golog.DebugLevel
	case "info", "1":
		return golog.InfoLevel
	case "warn", "warning", "2":
		return golog.WarnLevel
	case "error", "3":
		return golog.ErrorLevel
	case "fatal", "4":
		return golog.FatalLevel
	default:
		panic(fmt.Sprintf("💥💥 ERROR: invalid LOG_LEVEL %q (accepted: debug, info, warn, error, fatal or 0-4)", val))
	}
}

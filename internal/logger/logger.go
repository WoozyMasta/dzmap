// Package logger provides a configured zerolog instance.
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger holds configuration options for the application logger.
type Logger struct {
	//nolint:staticcheck // allow duplicate struct tags
	Level string `long:"log-level"   env:"LOG_LEVEL"  description:"Log level"  default:"info" choice:"trace" choice:"debug" choice:"info" choice:"warn" choice:"error" `
	//nolint:staticcheck // allow duplicate struct tags
	Format string `long:"log-format" env:"LOG_FORMAT" description:"Log format" default:"console" choice:"json" choice:"console" `
}

// Setup initializes the global logger based on provided configuration.
// It configures the output format (JSON or Console) and the logging level.
func (l *Logger) Setup() {
	level, err := zerolog.ParseLevel(l.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)

	if l.Format == "json" {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	} else {
		output := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}

		// Check if Stderr is a character device (terminal).
		// If it's a pipe or file redirection, disable colors.
		stat, _ := os.Stderr.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			output.NoColor = true
		}

		log.Logger = log.Output(output)
	}
}

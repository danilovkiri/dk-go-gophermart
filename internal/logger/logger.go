// Package logger provides logging functionality.

package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// InitLog initializes a logger.
func InitLog() *zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339
	Logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	return &Logger
}

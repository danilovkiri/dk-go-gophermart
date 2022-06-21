package logger

import (
	"github.com/rs/zerolog"
	"os"
	"time"
)

func InitLog() *zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339
	Logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	return &Logger
}

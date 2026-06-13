package logger

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// New constructs a JSON structured logger writing to stdout at the given level.
// Output is always JSON so request logs satisfy the structured-logging requirement.
func New(level string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	lvl, err := zerolog.ParseLevel(strings.ToLower(strings.TrimSpace(level)))
	if err != nil || level == "" {
		lvl = zerolog.InfoLevel
	}

	return zerolog.New(os.Stdout).
		Level(lvl).
		With().
		Timestamp().
		Str("service", "gdcpay-task-api").
		Logger()
}

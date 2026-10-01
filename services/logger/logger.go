package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init initializes global Zerolog logger with colorful console formatting.
// If jsonFormat is true, it outputs raw JSON to os.Stdout.
// If jsonFormat is false, it uses zerolog.ConsoleWriter for human-friendly colorized logs.
func Init(jsonFormat bool) {
	zerolog.TimeFieldFormat = time.RFC3339

	if jsonFormat {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05",
			NoColor:    false,
		}).With().Timestamp().Logger()
	}
}

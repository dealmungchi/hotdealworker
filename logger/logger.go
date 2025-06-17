package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init() {
	levelStr := os.Getenv("LOG_LEVEL")
	level, err := zerolog.ParseLevel(levelStr)
	if err != nil || levelStr == "" {
		level = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(level)

	cw := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}

	log.Logger = zerolog.New(cw).With().Timestamp().Logger()

	log.Info().
		Str("level", level.String()).
		Msg("Logger initialized")
}

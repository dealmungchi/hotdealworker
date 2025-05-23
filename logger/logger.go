package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger represents a structured logger
type Logger struct {
	logger zerolog.Logger
}

// Fields represents log fields
type Fields map[string]interface{}

var (
	// Default is the default logger instance
	Default *Logger
)

// Init initializes the logger with the given configuration
func Init() {
	level := getLogLevel()

	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(level)

	// Create console writer for development
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	// Create logger
	logger := zerolog.New(output).With().Timestamp().Logger()

	Default = &Logger{logger: logger}

	Default.Info().
		Str("level", level.String()).
		Msg("Logger initialized")
}

// getLogLevel returns the log level from environment variable
func getLogLevel() zerolog.Level {
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr == "" {
		levelStr = os.Getenv("HOTDEAL_ENVIRONMENT")
		if levelStr == "production" {
			return zerolog.InfoLevel
		}
		return zerolog.DebugLevel
	}

	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		return zerolog.InfoLevel
	}
	return level
}

// WithContext creates a new logger with context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{logger: zerolog.Ctx(ctx).With().Logger()}
}

// WithFields creates a new logger with fields
func (l *Logger) WithFields(fields Fields) *Logger {
	newLogger := l.logger.With()
	for k, v := range fields {
		newLogger = newLogger.Interface(k, v)
	}
	return &Logger{logger: newLogger.Logger()}
}

// WithField creates a new logger with a single field
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{logger: l.logger.With().Interface(key, value).Logger()}
}

// Debug returns a debug event
func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// Info returns an info event
func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Warn returns a warn event
func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// Error returns an error event
func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

// Fatal returns a fatal event
func (l *Logger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}

// WithError adds an error to the logger
func (l *Logger) WithError(err error) *Logger {
	return &Logger{logger: l.logger.With().Err(err).Logger()}
}

// Global functions for backward compatibility

// Debug logs a debug message
func Debug(format string, v ...interface{}) {
	if Default == nil {
		Init()
	}
	Default.Debug().Msgf(format, v...)
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	if Default == nil {
		Init()
	}
	Default.Info().Msgf(format, v...)
}

// Warn logs a warning message
func Warn(format string, v ...interface{}) {
	if Default == nil {
		Init()
	}
	Default.Warn().Msgf(format, v...)
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	if Default == nil {
		Init()
	}
	Default.Error().Msgf(format, v...)
}

// Fatal logs a fatal message and exits
func Fatal(format string, v ...interface{}) {
	if Default == nil {
		Init()
	}
	Default.Fatal().Msgf(format, v...)
}

// IsDebugEnabled returns true if debug logging is enabled
func IsDebugEnabled() bool {
	if Default == nil {
		Init()
	}
	return Default.logger.GetLevel() <= zerolog.DebugLevel
}

// ForCrawler creates a logger for a specific crawler
func ForCrawler(crawlerName string) *Logger {
	if Default == nil {
		Init()
	}
	return Default.WithField("crawler", crawlerName)
}

// ForWorker creates a logger for the worker
func ForWorker() *Logger {
	if Default == nil {
		Init()
	}
	return Default.WithField("component", "worker")
}

// ForPublisher creates a logger for the publisher
func ForPublisher() *Logger {
	if Default == nil {
		Init()
	}
	return Default.WithField("component", "publisher")
}

// ForCache creates a logger for the cache
func ForCache() *Logger {
	if Default == nil {
		Init()
	}
	return Default.WithField("component", "cache")
}

// LogError is a convenience method for logging errors with context
func LogError(component string, err error, format string, v ...interface{}) {
	if Default == nil {
		Init()
	}
	msg := fmt.Sprintf(format, v...)
	Default.Error().
		Str("component", component).
		Err(err).
		Msg(msg)
}

// LogInfo is a convenience method for logging info with context
func LogInfo(component string, format string, v ...interface{}) {
	if Default == nil {
		Init()
	}
	msg := fmt.Sprintf(format, v...)
	Default.Info().
		Str("component", component).
		Msg(msg)
}

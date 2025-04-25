package logger

import (
	"os"
	"strings"
	"time"

	"sjsage522/hotdealworker/config"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

// Init initializes the zerolog logger
func Init() {
	// 환경 설정 가져오기
	cfg := config.GetConfig()

	// 시간 포맷 설정
	zerolog.TimeFieldFormat = time.RFC3339

	// 콘솔 출력 설정
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    cfg.Environment == "production",
	}

	// 로거 설정
	logger = zerolog.New(consoleWriter).
		With().
		Timestamp().
		Caller().
		Logger()

	// 로그 레벨 설정 - 환경 변수에서 직접 가져옴
	setLogLevel(cfg.LogLevel)

	// 글로벌 로거 설정
	log.Logger = logger

	// 로그 레벨 정보 출력
	Info("Logger initialized with level: %s", cfg.LogLevel)
}

// setLogLevel sets the log level based on a string
func setLogLevel(level string) {
	switch strings.ToLower(level) {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		// 기본값은 info
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// IsDebugEnabled returns true if debug logging is enabled
func IsDebugEnabled() bool {
	return zerolog.GlobalLevel() <= zerolog.DebugLevel
}

// Debug logs a debug message
func Debug(format string, v ...interface{}) {
	log.Debug().Msgf(format, v...)
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	log.Info().Msgf(format, v...)
}

// Warn logs a warning message
func Warn(format string, v ...interface{}) {
	log.Warn().Msgf(format, v...)
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	log.Error().Msgf(format, v...)
}

// Fatal logs a fatal message and exits
func Fatal(format string, v ...interface{}) {
	log.Fatal().Msgf(format, v...)
}

// LoggerInterface defines the interface for logger implementations
type LoggerInterface interface {
	LogError(crawlerName string, err error)
	LogInfo(format string, args ...interface{})
	Debug(format string, args ...interface{})
}

// Logger provides logging functionality
type Logger struct{}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{}
}

// LogError logs an error
func (l *Logger) LogError(crawlerName string, err error) {
	log.Error().
		Str("crawler", crawlerName).
		Err(err).
		Msg("Crawler error")
}

// LogInfo logs an informational message
func (l *Logger) LogInfo(format string, args ...interface{}) {
	log.Info().Msgf(format, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	log.Debug().Msgf(format, args...)
}

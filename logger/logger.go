package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"sjsage522/hotdealworker/config"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

// LogLevel represents the severity of a log message
type LogLevel int

const (
	// DEBUG level for detailed information
	DEBUG LogLevel = iota
	// INFO level for general operational information
	INFO
	// WARN level for warning messages
	WARN
	// ERROR level for error messages
	ERROR
	// FATAL level for critical errors that cause the program to exit
	FATAL
)

var (
	// CurrentLevel is the current log level
	CurrentLevel LogLevel = DEBUG
	// TimeFormat is the format used for timestamps
	TimeFormat = time.RFC3339
)

// SetLevel sets the current log level
func SetLevel(level LogLevel) {
	CurrentLevel = level
}

// SetLevelFromString sets the log level from a string
func SetLevelFromString(level string) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		CurrentLevel = DEBUG
	case "INFO":
		CurrentLevel = INFO
	case "WARN":
		CurrentLevel = WARN
	case "ERROR":
		CurrentLevel = ERROR
	case "FATAL":
		CurrentLevel = FATAL
	default:
		CurrentLevel = INFO
	}
}

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
	if CurrentLevel <= DEBUG {
		logWithCaller(DEBUG, format, v...)
	}
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	if CurrentLevel <= INFO {
		logWithCaller(INFO, format, v...)
	}
}

// Warn logs a warning message
func Warn(format string, v ...interface{}) {
	if CurrentLevel <= WARN {
		logWithCaller(WARN, format, v...)
	}
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	if CurrentLevel <= ERROR {
		logWithCaller(ERROR, format, v...)
	}
}

// Fatal logs a fatal message and exits
func Fatal(format string, v ...interface{}) {
	if CurrentLevel <= FATAL {
		logWithCaller(FATAL, format, v...)
		os.Exit(1)
	}
}

// logWithCaller logs a message with caller information
func logWithCaller(level LogLevel, format string, v ...interface{}) {
	// Get caller information (skip 2 frames to get the actual caller)
	_, file, line, ok := runtime.Caller(2)
	callerInfo := "unknown"
	if ok {
		// Extract just the filename and directory
		file = filepath.Base(filepath.Dir(file)) + "/" + filepath.Base(file)
		callerInfo = fmt.Sprintf("%s:%d", file, line)
	}

	// Format the message
	message := fmt.Sprintf(format, v...)

	// Get the level name
	levelName := getLevelName(level)

	// Format the timestamp
	timestamp := time.Now().Format(TimeFormat)

	// Print the log message with caller information
	fmt.Fprintf(os.Stderr, "%s %s %s > %s\n", timestamp, levelName, callerInfo, message)
}

// getLevelName returns the string representation of a log level
func getLevelName(level LogLevel) string {
	switch level {
	case DEBUG:
		return "DBG"
	case INFO:
		return "INF"
	case WARN:
		return "WRN"
	case ERROR:
		return "ERR"
	case FATAL:
		return "FTL"
	default:
		return "???"
	}
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

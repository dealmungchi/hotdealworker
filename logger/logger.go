package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

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
	CurrentLevel LogLevel = DEBUG // 기본값을 DEBUG로 설정
	// TimeFormat is the format used for timestamps
	TimeFormat = time.RFC3339
)

// Init initializes the logger
func Init() {
	// 환경 변수에서 로그 레벨 가져오기
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		SetLevelFromString(logLevel)
	}

	// 로그 레벨 정보 출력
	Info("Logger initialized with level: %s", getLevelString(CurrentLevel))
}

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

// IsDebugEnabled returns true if debug logging is enabled
func IsDebugEnabled() bool {
	return CurrentLevel <= DEBUG
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

// getLevelString returns the full string representation of a log level
func getLevelString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
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

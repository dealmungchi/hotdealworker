package helpers

import (
	"fmt"
	"log"
	"os"
	"time"
)

// LoggerInterface defines the interface for logger implementations
type LoggerInterface interface {
	LogError(crawlerName string, err error)
	LogInfo(format string, args ...interface{})
}

// Logger provides logging functionality
type Logger struct {
	errorFile string
}

// NewLogger creates a new logger instance
func NewLogger(errorFile string) *Logger {
	return &Logger{
		errorFile: errorFile,
	}
}

// LogError logs an error to a file with crawler name and timestamp
func (l *Logger) LogError(crawlerName string, err error) {
	f, fileErr := os.OpenFile(l.errorFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if fileErr != nil {
		log.Printf("파일 열기 오류: %v\n", fileErr)
		return
	}
	defer f.Close()
	
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString(fmt.Sprintf("[%s] [%s] %s\n", timestamp, crawlerName, err.Error()))
}

// LogInfo logs an informational message
func (l *Logger) LogInfo(format string, args ...interface{}) {
	log.Printf(format, args...)
}
package errors

import (
	"fmt"
	"time"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// ErrorTypeNetwork represents network-related errors
	ErrorTypeNetwork ErrorType = "network"
	// ErrorTypeParsing represents HTML parsing errors
	ErrorTypeParsing ErrorType = "parsing"
	// ErrorTypeRateLimit represents rate limiting errors
	ErrorTypeRateLimit ErrorType = "rate_limit"
	// ErrorTypeCache represents cache-related errors
	ErrorTypeCache ErrorType = "cache"
	// ErrorTypePublisher represents publisher-related errors
	ErrorTypePublisher ErrorType = "publisher"
	// ErrorTypeValidation represents validation errors
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeConfiguration represents configuration errors
	ErrorTypeConfiguration ErrorType = "configuration"
)

// CrawlerError represents a crawler-specific error
type CrawlerError struct {
	Type     ErrorType
	Provider string
	Message  string
	Err      error
	Time     time.Time
}

// Error implements the error interface
func (e *CrawlerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s - %v", e.Type, e.Provider, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Provider, e.Message)
}

// Unwrap returns the underlying error
func (e *CrawlerError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true if the error is retryable
func (e *CrawlerError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeNetwork:
		return true
	case ErrorTypeRateLimit:
		return false
	case ErrorTypeParsing:
		return false
	default:
		return false
	}
}

// New creates a new CrawlerError
func New(errType ErrorType, provider, message string, err error) *CrawlerError {
	return &CrawlerError{
		Type:     errType,
		Provider: provider,
		Message:  message,
		Err:      err,
		Time:     time.Now(),
	}
}

// NewNetwork creates a new network error
func NewNetwork(provider, message string, err error) *CrawlerError {
	return New(ErrorTypeNetwork, provider, message, err)
}

// NewParsing creates a new parsing error
func NewParsing(provider, message string, err error) *CrawlerError {
	return New(ErrorTypeParsing, provider, message, err)
}

// NewRateLimit creates a new rate limit error
func NewRateLimit(provider string, duration time.Duration) *CrawlerError {
	message := fmt.Sprintf("rate limited for %v", duration)
	return New(ErrorTypeRateLimit, provider, message, nil)
}

// NewCache creates a new cache error
func NewCache(provider, message string, err error) *CrawlerError {
	return New(ErrorTypeCache, provider, message, err)
}

// NewPublisher creates a new publisher error
func NewPublisher(provider, message string, err error) *CrawlerError {
	return New(ErrorTypePublisher, provider, message, err)
}

// NewValidation creates a new validation error
func NewValidation(provider, message string) *CrawlerError {
	return New(ErrorTypeValidation, provider, message, nil)
}

// NewConfiguration creates a new configuration error
func NewConfiguration(message string, err error) *CrawlerError {
	return New(ErrorTypeConfiguration, "", message, err)
}

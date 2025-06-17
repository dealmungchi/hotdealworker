package publisher

// Publisher represents a service for publishing messages
type Publisher interface {
	// Publish publishes a message to a stream
	Publish(key string, message []byte) error

	// TrimStreams trims all streams to the configured maximum length
	TrimStreams() error

	// Close closes the publisher connection
	Close() error
}

// Note: For global publisher access, use services.GetPublisher() from the services package
// This avoids circular imports while providing centralized service management

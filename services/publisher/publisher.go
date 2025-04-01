package publisher

// Publisher represents a service for publishing messages
type Publisher interface {
	// Publish publishes a message to a stream
	Publish(message []byte) error

	// Close closes the publisher connection
	Close() error
}

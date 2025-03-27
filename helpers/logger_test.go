package helpers

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	// Create a temporary file for testing
	tmpFile := "./test_error.log"
	defer os.Remove(tmpFile) // Clean up after the test

	// Create a logger
	logger := NewLogger(tmpFile)

	// Log an error
	logger.LogError("TestCrawler", errors.New("test error"))

	// Check that the file was created and contains the error
	data, err := os.ReadFile(tmpFile)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "TestCrawler")
	assert.Contains(t, string(data), "test error")

	// Log an info message
	logger.LogInfo("Test info message: %s", "hello")

	// Info messages are logged to stdout, not the file
	// We can't easily test this without capturing stdout
}
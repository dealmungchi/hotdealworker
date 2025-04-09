package helpers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchWithRandomHeaders(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that headers are set
		assert.NotEmpty(t, r.Header.Get("User-Agent"))
		assert.NotEmpty(t, r.Header.Get("Accept"))
		assert.NotEmpty(t, r.Header.Get("Accept-Language"))
		assert.NotEmpty(t, r.Header.Get("Cookie"))
		assert.NotEmpty(t, r.Header.Get("referer"))

		// Send a response
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Hello, World!</body></html>"))
	}))
	defer server.Close()

	// Fetch the page
	reader, err := FetchWithRandomHeaders(server.URL)
	assert.NoError(t, err)

	// Read the response
	body, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Contains(t, string(body), "Hello, World!")
}

func TestFetchWithRandomHeadersNonUTF8(t *testing.T) {
	// Create a test server that returns a non-UTF8 response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send a response with a different charset
		w.Header().Set("Content-Type", "text/html; charset=iso-8859-1")
		w.WriteHeader(http.StatusOK)
		// This is "Hello, World!" in ISO-8859-1 encoding
		w.Write([]byte("<html><body>Hello, World!</body></html>"))
	}))
	defer server.Close()

	// Fetch the page
	reader, err := FetchWithRandomHeaders(server.URL)
	assert.NoError(t, err)

	// Read the response
	body, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Contains(t, string(body), "Hello, World!")
}

func TestFetchWithRandomHeadersError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Fetch the page
	_, err := FetchWithRandomHeaders(server.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 500")

	// Test with rate limiting
	serverRateLimited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer serverRateLimited.Close()

	// Fetch the page
	_, err = FetchWithRandomHeaders(serverRateLimited.URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limited")
}

func TestFetchWithRandomHeadersInvalidURL(t *testing.T) {
	// Fetch with an invalid URL
	_, err := FetchWithRandomHeaders("http://invalid.url.that.does.not.exist")
	assert.Error(t, err)
}

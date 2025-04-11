package crawler

import (
	"time"
)

// MockCacheService implements a simple in-memory cache for testing
type MockCacheService struct {
	cache map[string][]byte
}

func NewMockCacheService() *MockCacheService {
	return &MockCacheService{
		cache: make(map[string][]byte),
	}
}

func (m *MockCacheService) Get(key string) ([]byte, error) {
	if val, ok := m.cache[key]; ok {
		return val, nil
	}
	return nil, &mockError{message: "cache miss"}
}

func (m *MockCacheService) Set(key string, value []byte, expiration time.Duration) error {
	m.cache[key] = value
	return nil
}

func (m *MockCacheService) Delete(key string) error {
	delete(m.cache, key)
	return nil
}

type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}

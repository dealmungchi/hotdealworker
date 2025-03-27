package publisher

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// This test requires a running Redis instance
// If Redis is not available, the test will be skipped
func TestRedisPublisher(t *testing.T) {
	ctx := context.Background()
	publisher := NewRedisPublisher(ctx, "localhost:6379", 0)
	defer publisher.Close()

	// Create a subscriber to verify the message was published
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	defer client.Close()

	// Test if Redis is available
	_, err := client.Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis is not available, skipping test")
	}

	// Create a subscription
	pubsub := client.Subscribe(ctx, "test_channel")
	defer pubsub.Close()

	// Create a channel to receive messages
	messages := make(chan string, 1)
	
	// Start a goroutine to receive messages
	go func() {
		message, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			t.Errorf("Failed to receive message: %v", err)
			return
		}
		messages <- message.Payload
	}()

	// Give the subscription some time to be established
	time.Sleep(100 * time.Millisecond)

	// Publish a message
	err = publisher.Publish("test_channel", []byte("test_message"))
	assert.NoError(t, err)

	// Wait for the message to be received
	select {
	case msg := <-messages:
		// The message should be base64 encoded
		assert.Equal(t, "dGVzdF9tZXNzYWdl", msg) // base64 of "test_message"
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for message")
	}
}
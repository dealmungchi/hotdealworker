package publisher

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisPublisher(t *testing.T) {
	ctx := context.Background()
	publisher := NewRedisPublisher(ctx, "localhost:6379", 0, "test_stream_r")
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

	err = client.XGroupCreateMkStream(ctx, "test_stream_r", "test_group", "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		panic(err)
	}

	messages := make(chan string, 1)

	go func() {
		message, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Streams:  []string{"test_stream_r", ">"},
			Group:    "test_group",
			Consumer: "test_consumer",
			Block:    0,
		}).Result()
		assert.NoError(t, err)
		messages <- message[0].Messages[0].Values["b64_hotdeals"].(string)
	}()

	time.Sleep(100 * time.Millisecond)

	err = publisher.Publish([]byte("test_message"))
	assert.NoError(t, err)

	select {
	case msg := <-messages:
		// The message should be base64 encoded
		assert.Equal(t, "dGVzdF9tZXNzYWdl", msg) // base64 of "test_message"
	case <-time.After(1 * time.Second):
		t.Error("Timed out waiting for message")
	}
}

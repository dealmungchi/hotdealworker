package publisher

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisPublisher implements Publisher using Redis pub/sub
type RedisPublisher struct {
	client *redis.Client
	ctx    context.Context
	stream string
}

// NewRedisPublisher creates a new Redis publisher
func NewRedisPublisher(ctx context.Context, addr string, db int, stream string) *RedisPublisher {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   db,
	})

	return &RedisPublisher{
		client: client,
		ctx:    ctx,
		stream: stream,
	}
}

// Publish publishes a message to a Redis stream
// The message is base64 encoded before publishing
func (p *RedisPublisher) Publish(message []byte) error {
	// Make a copy of the message to ensure thread safety
	messageCopy := make([]byte, len(message))
	copy(messageCopy, message)

	// Base64 encode the message
	encodedMessage := base64.StdEncoding.EncodeToString(messageCopy)

	// Publish to Redis
	res, err := p.client.XAdd(p.ctx, &redis.XAddArgs{
		Stream: p.stream,
		Values: map[string]interface{}{
			"b64_hotdeals": encodedMessage,
		},
	}).Result()

	fmt.Println(res)

	return err
	// return p.client.Publish(p.ctx, stream, encodedMessage).Err()
}

// Close closes the Redis connection
func (p *RedisPublisher) Close() error {
	return p.client.Close()
}

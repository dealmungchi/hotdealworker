package publisher

import (
	"context"
	"encoding/base64"

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
func (p *RedisPublisher) Publish(key string, message []byte) error {
	// Base64 encode the message
	encodedMessage := base64.StdEncoding.EncodeToString(message)

	// Publish to Redis
	return p.client.XAdd(p.ctx, &redis.XAddArgs{
		Stream: p.stream,
		Values: map[string]interface{}{
			key: encodedMessage,
		},
	}).Err()
}

// Close closes the Redis connection
func (p *RedisPublisher) Close() error {
	return p.client.Close()
}

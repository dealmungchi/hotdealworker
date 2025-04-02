package publisher

import (
	"context"
	"encoding/base64"
	"strconv"

	"math/rand/v2"

	"github.com/redis/go-redis/v9"
)

// RedisPublisher implements Publisher using Redis pub/sub
type RedisPublisher struct {
	client       *redis.Client
	ctx          context.Context
	streamPrefix string
	streamCount  int
}

// NewRedisPublisher creates a new Redis publisher
func NewRedisPublisher(ctx context.Context, addr string, db int, streamPrefix string, streamCount int) *RedisPublisher {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   db,
	})

	return &RedisPublisher{
		client:       client,
		ctx:          ctx,
		streamPrefix: streamPrefix,
		streamCount:  streamCount,
	}
}

// Publish publishes a message to a Redis stream
// The message is base64 encoded before publishing
func (p *RedisPublisher) Publish(key string, message []byte) error {
	// Base64 encode the message
	encodedMessage := base64.StdEncoding.EncodeToString(message)

	// random stream name by streamCount
	// if streamCount is 10, stream name will be stream:0 ~ stream:9
	stream := p.streamPrefix + ":" + strconv.Itoa(rand.IntN((p.streamCount)))

	// Publish to Redis
	return p.client.XAdd(p.ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			key: encodedMessage,
		},
	}).Err()
}

// Close closes the Redis connection
func (p *RedisPublisher) Close() error {
	return p.client.Close()
}

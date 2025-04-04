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
	client          *redis.Client
	ctx             context.Context
	streamPrefix    string
	streamCount     int
	streamMaxLength int
}

// NewRedisPublisher creates a new Redis publisher
func NewRedisPublisher(ctx context.Context, addr string, db int, streamPrefix string, streamCount int, streamMaxLength int) *RedisPublisher {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   db,
	})

	return &RedisPublisher{
		client:          client,
		ctx:             ctx,
		streamPrefix:    streamPrefix,
		streamCount:     streamCount,
		streamMaxLength: streamMaxLength,
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

// TrimStreams trims all streams to the configured maximum length
func (p *RedisPublisher) TrimStreams() error {
	// Get all streams with the prefix
	pattern := p.streamPrefix + ":*"
	streams, err := p.client.Keys(p.ctx, pattern).Result()
	if err != nil {
		return err
	}

	// Trim each stream
	for _, stream := range streams {
		err := p.client.XTrimMaxLen(p.ctx, stream, int64(p.streamMaxLength)).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

// Close closes the Redis connection
func (p *RedisPublisher) Close() error {
	return p.client.Close()
}

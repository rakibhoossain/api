package buffers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisBuffer struct {
	name             string
	rdb              *redis.Client
	redisKey         string
	bufferCountKey   string
	batchSize        int
	chunkSize        int
	processingBuffer bool
}

func NewRedisBuffer(name string, rdb *redis.Client) *RedisBuffer {
	return &RedisBuffer{
		name:           name,
		rdb:            rdb,
		redisKey:       name + "-buffer",
		bufferCountKey: name + "-buffer:count",
		batchSize:      1000,
		chunkSize:      1000,
	}
}

// Add pushes the JSON representation of an item to the Redis list and increments the counter
func (b *RedisBuffer) Add(item interface{}) {
	ctx := context.Background()
	bytes, err := json.Marshal(item)
	if err != nil {
		log.Printf("[%s] Failed to marshal item: %v", b.name, err)
		return
	}

	pipeline := b.rdb.TxPipeline()
	pipeline.RPush(ctx, b.redisKey, string(bytes))
	pipeline.Incr(ctx, b.bufferCountKey)
	_, err = pipeline.Exec(ctx)
	if err != nil {
		log.Printf("[%s] Failed to add item to redis: %v", b.name, err)
	}
}

// FetchItems gets up to batchSize items from the Redis list without removing them
func (b *RedisBuffer) FetchItems(ctx context.Context) ([]string, error) {
	return b.rdb.LRange(ctx, b.redisKey, 0, int64(b.batchSize-1)).Result()
}

// RemoveProcessedItems trims the items that were successfully processed from the list
func (b *RedisBuffer) RemoveProcessedItems(ctx context.Context, count int64) error {
	pipeline := b.rdb.TxPipeline()
	pipeline.LTrim(ctx, b.redisKey, count, -1)
	pipeline.DecrBy(ctx, b.bufferCountKey, count)
	_, err := pipeline.Exec(ctx)
	return err
}

func (b *RedisBuffer) TryFlush(processFunc func(ctx context.Context, items []string) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	lockKey := "lock:" + b.name
	lockId := uuid.New().String()

	// Try to acquire lock
	acquired, err := b.rdb.SetNX(ctx, lockKey, lockId, 60*time.Second).Result()
	if err != nil || !acquired {
		return nil // skip silently if we can't get lock, another node is processing
	}

	defer func() {
		// Release lock safely via Lua script
		script := `
			if redis.call("get", KEYS[1]) == ARGV[1] then
				return redis.call("del", KEYS[1])
			else
				return 0
			end
		`
		b.rdb.Eval(context.Background(), script, []string{lockKey}, lockId)
	}()

	items, err := b.FetchItems(ctx)
	if err != nil || len(items) == 0 {
		return err
	}

	log.Printf("[%s] Flushing %d items", b.name, len(items))
	err = processFunc(ctx, items)
	if err != nil {
		log.Printf("[%s] Process func failed: %v", b.name, err)
		return err
	}

	return b.RemoveProcessedItems(ctx, int64(len(items)))
}

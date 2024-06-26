package extredis

import (
	"context"
	"encoding/hex"
	"errors"

	redis "github.com/redis/go-redis/v9"
)

var (
	ErrCacheMiss = errors.New("big_cache: key/value is not cached")
)

type BigCache struct {
	pool *RedisClientConnPool
}

func InstallBigCache(pool *RedisClientConnPool) *BigCache {
	return &BigCache{pool: pool}
}

func (impl *BigCache) Keys(
	ctx context.Context,
	cursor uint64,
	pattern string,
	batchSize int64,
) ([]string, uint64, bool, error) {
	var keys []string
	var more bool
	var err error
	// A full iteration starts when the cursor is set to 0,
	// and terminates when the cursor returned by the server is 0.
	keys, cursor, err = impl.pool.client.Scan(ctx, cursor, pattern, batchSize).Result()
	if err == nil {
		if cursor == 0 {
			more = false
		} else {
			more = true
		}
	}
	return keys, cursor, more, err
}

func (impl *BigCache) Set(
	ctx context.Context,
	item *Item,
) error {
	return impl.set(ctx, item)
}

func (impl *BigCache) set(
	ctx context.Context,
	item *Item,
) error {
	b, err := Marshal(item.Value())
	if err != nil {
		return err
	}
	return impl.pool.client.Set(ctx, item.Key(), hex.EncodeToString(b), item.TTL()).Err()
}

func (impl *BigCache) SetString(
	ctx context.Context,
	item *Item,
) error {
	return impl.pool.client.Set(ctx, item.Key(), item.Value().(string), item.TTL()).Err()
}

func (impl *BigCache) SetInt(
	ctx context.Context,
	item *Item,
) error {
	return impl.pool.client.Set(ctx, item.Key(), item.Value().(int), item.TTL()).Err()
}

func (impl *BigCache) SetInt64(
	ctx context.Context,
	item *Item,
) error {
	return impl.pool.client.Set(ctx, item.Key(), item.Value().(int64), item.TTL()).Err()
}

func (impl *BigCache) Exist(
	ctx context.Context,
	key string,
) bool {
	_, err := impl.getValue(ctx, key)
	return err == nil
}

func (impl *BigCache) Get(
	ctx context.Context,
	key string,
	value interface{},
) error {
	return impl.get(ctx, key, value)
}

func (impl *BigCache) get(
	ctx context.Context,
	key string,
	value interface{},
) error {
	b, err := impl.getValue(ctx, key)
	if err != nil {
		return err
	}
	return Unmarshal(b, value)
}

func (impl *BigCache) getValue(
	ctx context.Context,
	key string,
) ([]byte, error) {
	ret, err := impl.pool.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrCacheMiss
		}
		return nil, err
	}
	b, _ := hex.DecodeString(ret)
	return b, nil
}

func (impl *BigCache) GetString(
	ctx context.Context,
	key string,
	value *string,
) error {
	*value = impl.pool.client.Get(ctx, key).Val()
	return nil
}

func (impl *BigCache) GetInt(
	ctx context.Context,
	key string,
	value *int,
) error {
	var err error
	*value, err = impl.pool.client.Get(ctx, key).Int()
	return err
}

func (impl *BigCache) GetInt64(
	ctx context.Context,
	key string,
	value *int64,
) error {
	var err error
	*value, err = impl.pool.client.Get(ctx, key).Int64()
	return err
}

func (impl *BigCache) Incr(
	ctx context.Context,
	key string,
) error {
	_, err := impl.pool.client.Incr(ctx, key).Result()
	return err
}

func (impl *BigCache) Decr(
	ctx context.Context,
	key string,
) error {
	_, err := impl.pool.client.Decr(ctx, key).Result()
	return err
}

// SafeDecr is like Decr but makes sure the value never goes below zero.
func (impl *BigCache) SafeDecr(
	ctx context.Context,
	key string,
) error {
	_, err := safeDecrLuaScript.Run(ctx, impl.pool.client, []string{key}).Result()
	return err
}

func (impl *BigCache) Del(
	ctx context.Context,
	key string,
) error {
	_, err := impl.pool.client.Del(ctx, key).Result()
	return err
}

package redisx

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const unlockLua = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`

type Client struct {
	rdb *redis.Client
}

func Open(addr string) *Client {
	return &Client{rdb: redis.NewClient(&redis.Options{
		Addr:         addr,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})}
}

func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) IncrWindow(ctx context.Context, key string, window time.Duration) (int64, error) {
	n, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 {
		_ = c.rdb.Expire(ctx, key, window).Err()
	}
	return n, nil
}

// TryLock implements booking.DistLock with SET NX PX and token-compared release.
func (c *Client) TryLock(ctx context.Context, key string, ttl time.Duration) (bool, func(), error) {
	if ttl <= 0 {
		ttl = 8 * time.Second
	}
	token := uuid.NewString()
	ms := ttl.Milliseconds()
	if ms < 1 {
		ms = 1
	}
	res, err := c.rdb.Do(ctx, "SET", key, token, "NX", "PX", ms).Result()
	if err == redis.Nil || res == nil {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	var once sync.Once
	unlock := func() {
		once.Do(func() {
			uctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = c.rdb.Eval(uctx, unlockLua, []string{key}, token).Err()
		})
	}
	return true, unlock, nil
}

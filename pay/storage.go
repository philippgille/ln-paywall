package pay

import (
	"github.com/go-redis/redis"
)

// RedisClient is a StorageClient implementation for Redis.
type RedisClient struct {
	c *redis.Client
}

// WasUsed checks if the preimage was used for a previous payment already.
func (c RedisClient) WasUsed(preimage string) (bool, error) {
	_, err := c.c.Get(preimage).Result()
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

// SetUsed stores the information that a preimage has been used for a payment.
func (c RedisClient) SetUsed(preimage string) error {
	err := c.c.Set(preimage, true, 0).Err()
	if err != nil {
		return err
	}
	return nil
}

// RedisOptions are the options for the Redis DB.
type RedisOptions struct {
	Address string
	// Password for the Redis server, optional ("" by default)
	Password string
	// DB to use, optional (0 by default)
	DB int
}

// NewRedisClient creates a new RedisClient.
func NewRedisClient(redisOptions RedisOptions) RedisClient {
	return RedisClient{
		c: redis.NewClient(&redis.Options{
			Addr:     redisOptions.Address,
			Password: redisOptions.Password,
			DB:       redisOptions.DB,
		}),
	}
}

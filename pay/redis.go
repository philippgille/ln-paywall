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
	// Address of the Redis server, including the port, optional ("localhost:6379" by default)
	Address string
	// Password for the Redis server, optional ("" by default)
	Password string
	// DB to use, optional (0 by default)
	DB int
}

// DefaultRedisOptions is a RedisOptions object with default values.
// Address: "localhost:6379", Password: "", DB: 0
var DefaultRedisOptions = RedisOptions{
	Address: "localhost:6379",
	// No need to set Password or DB, since their Go zero values are fine for that
}

// NewRedisClient creates a new RedisClient.
func NewRedisClient(redisOptions RedisOptions) RedisClient {
	// Set default values
	if redisOptions.Address == "" {
		redisOptions.Address = "localhost:6379"
	}
	return RedisClient{
		c: redis.NewClient(&redis.Options{
			Addr:     redisOptions.Address,
			Password: redisOptions.Password,
			DB:       redisOptions.DB,
		}),
	}
}

package wall_test

import (
	"testing"

	"github.com/philippgille/ln-paywall/wall"
)

// TestRedisClient tests if the RedisClient struct implements the StorageClient interface.
// This doesn't happen at runtime, but at compile time.
func TestRedisClient(t *testing.T) {
	t.SkipNow()
	invoiceOptions := wall.InvoiceOptions{}
	lndOptions := wall.LNDoptions{}
	redisClient := wall.RedisClient{}
	wall.NewHandlerFuncMiddleware(invoiceOptions, lndOptions, redisClient)
	wall.NewHandlerMiddleware(invoiceOptions, lndOptions, redisClient)
	wall.NewGinMiddleware(invoiceOptions, lndOptions, redisClient)
}

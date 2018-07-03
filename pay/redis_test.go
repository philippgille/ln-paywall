package pay_test

import (
	"testing"

	"github.com/philippgille/ln-paywall/pay"
)

// TestRedisClient tests if the RedisClient struct implements the StorageClient interface.
// This doesn't happen at runtime, but at compile time.
func TestRedisClient(t *testing.T) {
	t.SkipNow()
	invoiceOptions := pay.InvoiceOptions{}
	lndOptions := pay.LNDoptions{}
	redisClient := pay.RedisClient{}
	pay.NewHandlerFuncMiddleware(invoiceOptions, lndOptions, redisClient)
	pay.NewHandlerMiddleware(invoiceOptions, lndOptions, redisClient)
	pay.NewGinMiddleware(invoiceOptions, lndOptions, redisClient)
}

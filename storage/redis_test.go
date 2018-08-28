package storage_test

import (
	"testing"

	"github.com/philippgille/ln-paywall/ln"
	"github.com/philippgille/ln-paywall/storage"
	"github.com/philippgille/ln-paywall/wall"
)

// TestRedisClient tests if the RedisClient struct implements the StorageClient interface.
// This doesn't happen at runtime, but at compile time.
func TestRedisClient(t *testing.T) {
	t.SkipNow()
	invoiceOptions := wall.InvoiceOptions{}
	lnClient := ln.LNDclient{}
	redisClient := storage.RedisClient{}
	wall.NewHandlerFuncMiddleware(invoiceOptions, lnClient, redisClient)
	wall.NewHandlerMiddleware(invoiceOptions, lnClient, redisClient)
	wall.NewGinMiddleware(invoiceOptions, lnClient, redisClient)
}

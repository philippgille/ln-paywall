package storage_test

import (
	"testing"

	"github.com/philippgille/ln-paywall/storage"
	"github.com/philippgille/ln-paywall/wall"
)

// TestBoltClient tests if the BoltClient struct implements the StorageClient interface.
// This doesn't happen at runtime, but at compile time.
func TestBoltClient(t *testing.T) {
	t.SkipNow()
	invoiceOptions := wall.InvoiceOptions{}
	lndOptions := wall.LNDoptions{}
	boltClient, _ := storage.NewBoltClient(storage.DefaultBoltOptions)
	wall.NewHandlerFuncMiddleware(invoiceOptions, lndOptions, boltClient)
	wall.NewHandlerMiddleware(invoiceOptions, lndOptions, boltClient)
	wall.NewGinMiddleware(invoiceOptions, lndOptions, boltClient)
	wall.NewEchoMiddleware(invoiceOptions, lndOptions, boltClient, nil)
}

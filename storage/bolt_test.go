package storage_test

import (
	"testing"

	"github.com/philippgille/ln-paywall/ln"
	"github.com/philippgille/ln-paywall/storage"
	"github.com/philippgille/ln-paywall/wall"
)

// TestBoltClient tests if the BoltClient struct implements the StorageClient interface.
// This doesn't happen at runtime, but at compile time.
func TestBoltClient(t *testing.T) {
	t.SkipNow()
	invoiceOptions := wall.InvoiceOptions{}
	lnClient := ln.LNDclient{}
	boltClient, _ := storage.NewBoltClient(storage.DefaultBoltOptions)
	wall.NewHandlerFuncMiddleware(invoiceOptions, lnClient, boltClient)
	wall.NewHandlerMiddleware(invoiceOptions, lnClient, boltClient)
	wall.NewGinMiddleware(invoiceOptions, lnClient, boltClient)
	wall.NewEchoMiddleware(invoiceOptions, lnClient, boltClient, nil)
}

package pay_test

import (
	"testing"

	"github.com/philippgille/ln-paywall/pay"
)

// TestBoltClient tests if the BoltClient struct implements the StorageClient interface.
// This doesn't happen at runtime, but at compile time.
func TestBoltClient(t *testing.T) {
	t.SkipNow()
	invoiceOptions := pay.InvoiceOptions{}
	lndOptions := pay.LNDoptions{}
	boltClient, _ := pay.NewBoltClient(pay.DefaultBoltOptions)
	pay.NewHandlerFuncMiddleware(invoiceOptions, lndOptions, boltClient)
	pay.NewHandlerMiddleware(invoiceOptions, lndOptions, boltClient)
	pay.NewGinMiddleware(invoiceOptions, lndOptions, boltClient)
	pay.NewEchoMiddleware(invoiceOptions, lndOptions, boltClient, nil)
}

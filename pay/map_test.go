package pay_test

import (
	"testing"

	"github.com/philippgille/ln-paywall/pay"
)

// TestGoMap tests if the GoMap struct implements the StorageClient interface.
// This doesn't happen at runtime, but at compile time.
func TestGoMap(t *testing.T) {
	t.SkipNow()
	invoiceOptions := pay.InvoiceOptions{}
	lndOptions := pay.LNDoptions{}
	goMap := pay.GoMap{}
	pay.NewHandlerFuncMiddleware(invoiceOptions, lndOptions, goMap)
	pay.NewHandlerMiddleware(invoiceOptions, lndOptions, goMap)
	pay.NewGinMiddleware(invoiceOptions, lndOptions, goMap)
}

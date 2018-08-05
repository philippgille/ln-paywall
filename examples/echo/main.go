package main

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/philippgille/ln-paywall/pay"
)

func main() {
	e := echo.New()

	// Configure and use middleware
	invoiceOptions := pay.DefaultInvoiceOptions // Price: 1 Satoshi; Memo: "API call"
	lndOptions := pay.DefaultLNDoptions         // Address: "localhost:10009", CertFile: "tls.cert", MacaroonFile: "invoice.macaroon"
	storageClient := pay.NewGoMap()
	e.Use(pay.NewEchoMiddleware(invoiceOptions, lndOptions, storageClient, nil))

	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	e.Logger.Fatal(e.Start(":8080")) // Start server
}

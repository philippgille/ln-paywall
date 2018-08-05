package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/philippgille/ln-paywall/pay"
)

func main() {
	r := gin.Default()

	// Configure and use middleware
	invoiceOptions := pay.DefaultInvoiceOptions // Price: 1 Satoshi; Memo: "API call"
	lndOptions := pay.DefaultLNDoptions         // Address: "localhost:10009", CertFile: "tls.cert", MacaroonFile: "invoice.macaroon"
	storageClient := pay.NewGoMap()
	r.Use(pay.NewGinMiddleware(invoiceOptions, lndOptions, storageClient))

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.Run() // listen and serve on 0.0.0.0:8080
}

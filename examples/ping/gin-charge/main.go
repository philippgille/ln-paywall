package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/philippgille/ln-paywall/ln"
	"github.com/philippgille/ln-paywall/storage"
	"github.com/philippgille/ln-paywall/wall"
)

func main() {
	r := gin.Default()

	// Configure middleware
	invoiceOptions := wall.DefaultInvoiceOptions // Price: 1 Satoshi; Memo: "API call"
	chargeOptions := ln.ChargeOptions{           // Address: "http://localhost:9112"
		APItoken: "secret",
	}
	storageClient := storage.NewGoMap() // Local in-memory cache
	lnClient, err := ln.NewChargeClient(chargeOptions)
	if err != nil {
		panic(err)
	}
	// Use middleware
	r.Use(wall.NewGinMiddleware(invoiceOptions, lnClient, storageClient))

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.Run() // Listen and serve on 0.0.0.0:8080
}

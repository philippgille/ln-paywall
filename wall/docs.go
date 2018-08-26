/*
Package wall contains all paywall-related code.

This is the package you need to use for creating a middleware for one of the supported handlers / routers / frameworks.
For creating a middleware you only need to call one of the provided factory functions,
but all functions require a storage client (an implementation of the wall.StorageClient interface) as parameter.
You can either pick one from the storage package (https://www.godoc.org/github.com/philippgille/ln-paywall/storage), or implement your own.

Usage

Here's one example of a web service implemented with Gin.
For more examples check out the "examples" directory in the GitHub repository of this package.

	package main

	import (
		"net/http"

		"github.com/gin-gonic/gin"
		"github.com/philippgille/ln-paywall/storage"
		"github.com/philippgille/ln-paywall/wall"
	)

	func main() {
		r := gin.Default()

		// Configure and use middleware
		invoiceOptions := wall.DefaultInvoiceOptions // Price: 1 Satoshi; Memo: "API call"
		lndOptions := wall.DefaultLNDoptions         // Address: "localhost:10009", CertFile: "tls.cert", MacaroonFile: "invoice.macaroon"
		storageClient := storage.NewGoMap()          // Local in-memory cache
		r.Use(wall.NewGinMiddleware(invoiceOptions, lndOptions, storageClient))

		r.GET("/ping", func(c *gin.Context) {
			c.String(http.StatusOK, "pong")
		})

		r.Run() // Listen and serve on 0.0.0.0:8080
	}
*/
package wall

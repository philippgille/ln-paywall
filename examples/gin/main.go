package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/philippgille/ln-paywall/pay"
)

func main() {
	r := gin.Default()

	// Configure and use middleware
	invoiceOptions := pay.InvoiceOptions{
		Amount: 1,
		Memo:   "Ping API call",
	}
	lndOptions := pay.LNDoptions{
		Address:      "123.123.123.123:10009",
		CertFile:     "tls.cert",
		MacaroonFile: "invoice.macaroon",
	}
	redisClient := pay.NewRedisClient(pay.DefaultRedisOptions) // Connects to localhost:6379
	r.Use(pay.NewGinMiddleware(invoiceOptions, lndOptions, redisClient))

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.Run() // listen and serve on 0.0.0.0:8080
}

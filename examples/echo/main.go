package main

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/philippgille/ln-paywall/pay"
)

func main() {
	e := echo.New()

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
	e.Use(pay.NewEchoMiddleware(invoiceOptions, lndOptions, redisClient, nil))

	e.GET("/ping", func(c echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})

	e.Logger.Fatal(e.Start(":8080")) // Start server
}

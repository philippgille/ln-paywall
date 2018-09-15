package main

import (
	"flag"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/philippgille/ln-paywall/ln"
	"github.com/philippgille/ln-paywall/storage"
	"github.com/philippgille/ln-paywall/wall"
	qrcode "github.com/skip2/go-qrcode"
)

var lndAddress = flag.String("addr", "localhost:10009", "Address of the lnd node (including gRPC port)")
var dataDir = flag.String("dataDir", "data/", "Relative path to the data directory, where tls.cert and invoice.macaroon are located")
var price = flag.Int64("price", 1000, "Price of one request in Satoshis (at an exchange rate of $1,000 for 1 BTC 1000 Satoshis would be $0.01)")

func main() {
	flag.Parse()

	// Make sure the path to the data directory ends with "/"
	dataDirSuffixed := *dataDir
	if !strings.HasSuffix(dataDirSuffixed, "/") {
		dataDirSuffixed += "/"
	}

	r := gin.Default()

	// Configure middleware

	// Invoice
	invoiceOptions := wall.InvoiceOptions{
		Memo:  "QR code generation API call",
		Price: *price,
	}

	// LN client
	lndOptions := ln.LNDoptions{
		Address:      *lndAddress,
		CertFile:     dataDirSuffixed + "tls.cert",
		MacaroonFile: dataDirSuffixed + "invoice.macaroon",
	}
	lnClient, err := ln.NewLNDclient(lndOptions)
	if err != nil {
		panic(err)
	}

	// Storage
	boltOptions := storage.BoltOptions{
		Path: dataDirSuffixed + "qr-code.db",
	}
	storageClient, err := storage.NewBoltClient(boltOptions)
	if err != nil {
		panic(err)
	}

	// Use middleware
	r.Use(wall.NewGinMiddleware(invoiceOptions, lnClient, storageClient))

	r.GET("/qr", qrHandler)

	r.Run() // Listen and serve on 0.0.0.0:8080
}

func qrHandler(c *gin.Context) {
	data := c.Query("data")
	if data == "" {
		c.String(http.StatusBadRequest, "The query parameter \"data\" is missing")
		c.Abort()
	} else {
		qrBytes, err := qrcode.Encode(data, qrcode.Medium, 256)
		if err != nil {
			c.String(http.StatusInternalServerError, "There was an error encoding the data as QR code")
			c.Abort()
		} else {
			c.Data(http.StatusOK, "image/png", qrBytes)
		}
	}
}

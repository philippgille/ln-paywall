package wall

import (
	"log"
	"os"
)

// stdOutLogger logs to stdout, while the default log package loggers log to stderr.
var stdOutLogger = log.New(os.Stdout, "", log.LstdFlags)

// InvoiceOptions are the options for an invoice.
type InvoiceOptions struct {
	// Amount of Satoshis you want to have paid for one API call.
	// Values below 1 are automatically changed to the default value.
	// Optional (1 by default).
	Price int64
	// Note to be shown on the invoice,
	// for example: "API call to api.example.com".
	// Optional ("" by default).
	Memo string
}

// DefaultInvoiceOptions provides default values for InvoiceOptions.
var DefaultInvoiceOptions = InvoiceOptions{
	Price: 1,
	Memo:  "API call",
}

// LNDoptions are the options for the connection to the lnd node.
type LNDoptions struct {
	// Address of your LND node, including the port.
	// Optional ("localhost:10009" by default).
	Address string
	// Path to the "tls.cert" file that your LND node uses.
	// Optional ("tls.cert" by default).
	CertFile string
	// Path to the "invoice.macaroon" file that your LND node uses.
	// Optional ("invoice.macaroon" by default).
	MacaroonFile string
}

// DefaultLNDoptions provides default values for LNDoptions.
var DefaultLNDoptions = LNDoptions{
	Address:      "localhost:10009",
	CertFile:     "tls.cert",
	MacaroonFile: "invoice.macaroon",
}

// StorageClient is an abstraction for different storage client implementations.
// A storage client must only be able to check if a preimage was already used for a payment bofore
// and to store a preimage that was used before.
type StorageClient interface {
	WasUsed(string) (bool, error)
	SetUsed(string) error
}

// LNclient is an abstraction of a client that connects to a Lightning Network node implementation (like lnd, c-lightning and eclair)
// and provides the methods required by the paywall.
type LNclient interface {
	GenerateInvoice(int64, string) (string, error)
	CheckInvoice(string) (bool, error)
}

// handlePreimage does four things:
// 1) Checks if the preimage was already used as a payment proof before.
// 2) Checks if the preimage corresponds to an existing invoice on the connected lnd.
// 3) Checks if the corresponding invoice was settled.
// 4) Store the preimage to the storage for future checks.
// Returns false and an empty error if the preimage was already used or if the corresponding invoice isn't settled.
// Returns true and an empty error if everythings's fine.
func handlePreimage(preimage string, storageClient StorageClient, lndClient LNclient) (bool, error) {
	// Check if it was already used before
	wasUsed, err := storageClient.WasUsed(preimage)
	if err != nil {
		return false, err
	}
	if wasUsed {
		// Key was found, which means the payment was already used for an API call.
		return false, nil
	}

	// Check if a corresponding invoice exists and is settled
	settled, err := lndClient.CheckInvoice(preimage)
	if err != nil {
		return false, err
	}
	if !settled {
		return false, nil
	}

	// Key not found, so it wasn't used before.
	// Insert key for future checks.
	err = storageClient.SetUsed(preimage)
	if err != nil {
		return true, err
	}
	return true, nil
}

func assignDefaultValues(invoiceOptions InvoiceOptions, lndOptions LNDoptions) (InvoiceOptions, LNDoptions) {
	// InvoiceOptions
	if invoiceOptions.Price <= 0 {
		invoiceOptions.Price = DefaultInvoiceOptions.Price
	}
	// Empty Memo is okay.

	// LNDoptions
	if lndOptions.Address == "" {
		lndOptions.Address = DefaultLNDoptions.Address
	}
	if lndOptions.CertFile == "" {
		lndOptions.CertFile = DefaultLNDoptions.CertFile
	}
	if lndOptions.MacaroonFile == "" {
		lndOptions.MacaroonFile = DefaultLNDoptions.MacaroonFile
	}

	return invoiceOptions, lndOptions
}

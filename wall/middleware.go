package wall

import (
	"log"
	"os"
	"reflect"
	"strings"
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
// 2) Checks if the preimage corresponds to an existing invoice on the connected LN node.
// 3) Checks if the corresponding invoice was settled.
// 4) Store the preimage to the storage for future checks.
// Returns a string and an error.
// The string contains detailed info about the result in case the preimage is invalid.
// The error is only non-nil if an error occurs during the check (like the LN node can't be reached).
// The preimage is only valid if the string is empty and the error is nil.
func handlePreimage(preimage string, storageClient StorageClient, lnClient LNclient) (string, error) {
	// Check if it was already used before
	wasUsed, err := storageClient.WasUsed(preimage)
	if err != nil {
		return "", err
	}
	if wasUsed {
		// Key was found, which means the payment was already used for an API call.
		return "The provided preimage was already used in a previous request", nil
	}

	// Check if a corresponding invoice exists and is settled
	settled, err := lnClient.CheckInvoice(preimage)
	if err != nil {
		// Returning a non-nil error leads to an "internal server error", but in some cases it's a "bad request".
		// TODO: Both checks should be done in a more robust and elegant way
		if reflect.TypeOf(err).Name() == "CorruptInputError" {
			return "The provided preimage contains invalid Base64 characters", nil
		} else if strings.Contains(err.Error(), "unable to locate invoice") {
			return "No corresponding invoice was found for the provided preimage", nil
		} else {
			return "", err
		}
	}
	if !settled {
		return "You somehow obtained the preimage of the invoice, but the invoice is not settled yet", nil
	}

	// Key not found, so it wasn't used before.
	// Insert key for future checks.
	err = storageClient.SetUsed(preimage)
	if err != nil {
		return "", err
	}
	return "", nil
}

func assignDefaultValues(invoiceOptions InvoiceOptions) InvoiceOptions {
	// InvoiceOptions
	if invoiceOptions.Price <= 0 {
		invoiceOptions.Price = DefaultInvoiceOptions.Price
	}
	// Empty Memo is okay.

	return invoiceOptions
}

package ln

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/lightningnetwork/lnd/lnrpc"
)

// LNDclient is an implementation of the wall.Client interface for the lnd Lightning Network node implementation.
type LNDclient struct {
	lndClient lnrpc.LightningClient
	ctx       context.Context
	conn      *grpc.ClientConn
}

// GenerateInvoice generates an invoice with the given price and memo.
func (c LNDclient) GenerateInvoice(amount int64, memo string) (string, error) {
	// Create the request and send it
	invoice := lnrpc.Invoice{
		Memo:  memo,
		Value: amount,
	}
	log.Println("Creating invoice for a new API request")
	res, err := c.lndClient.AddInvoice(c.ctx, &invoice)
	if err != nil {
		return "", err
	}

	return res.GetPaymentRequest(), nil
}

// CheckInvoice takes a hex encoded preimage and checks if the corresponding invoice was settled.
// An error is returned if the preimage isn't properly encoded or if no corresponding invoice was found.
// False is returned if the invoice isn't settled.
func (c LNDclient) CheckInvoice(preimageHex string) (bool, error) {
	preimageHashHex, err := HashPreimage(preimageHex)
	if err != nil {
		return false, err
	}
	// Ignore the error because the reverse (encoding) was just done previously, so this must work
	plainHash, _ := hex.DecodeString(preimageHashHex)

	log.Printf("Checking invoice for hash %v\n", preimageHashHex)

	// Get the invoice for that hash
	paymentHash := lnrpc.PaymentHash{
		RHash: plainHash,
		// Hex encoded, must be exactly 32 byte
		RHashStr: preimageHashHex,
	}
	invoice, err := c.lndClient.LookupInvoice(c.ctx, &paymentHash)
	if err != nil {
		return false, err
	}

	// Check if invoice was settled
	if !invoice.GetSettled() {
		return false, nil
	}
	return true, nil
}

// NewLNDclient creates a new LNDclient instance.
func NewLNDclient(lndOptions LNDoptions) (LNDclient, error) {
	result := LNDclient{}

	lndOptions = assignLNDdefaultValues(lndOptions)

	// Set up a connection to the server.
	creds, err := credentials.NewClientTLSFromFile(lndOptions.CertFile, "")
	if err != nil {
		return result, err
	}
	conn, err := grpc.Dial(lndOptions.Address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return result, err
	}
	c := lnrpc.NewLightningClient(conn)

	// Add the macaroon to the outgoing context

	macaroon, err := ioutil.ReadFile(lndOptions.MacaroonFile)
	if err != nil {
		return result, err
	}
	// Value must be the hex representation of the file content
	macaroonHex := hex.EncodeToString(macaroon)
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "macaroon", macaroonHex)

	result = LNDclient{
		conn:      conn,
		ctx:       ctx,
		lndClient: c,
	}

	return result, nil
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

func assignLNDdefaultValues(lndOptions LNDoptions) LNDoptions {
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

	return lndOptions
}

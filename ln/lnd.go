package ln

import (
	"context"
	"encoding/hex"
	"errors"
	"io/ioutil"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/lightningnetwork/lnd/lnrpc"
)

// LNDclient is an implementation of the wall.LNClient and pay.LNClient interface
// for the lnd Lightning Network node implementation.
type LNDclient struct {
	lndClient lnrpc.LightningClient
	ctx       context.Context
	conn      *grpc.ClientConn
}

// GenerateInvoice generates an invoice with the given price and memo.
func (c LNDclient) GenerateInvoice(amount int64, memo string) (Invoice, error) {
	result := Invoice{}

	// Create the request and send it
	invoice := lnrpc.Invoice{
		Memo:  memo,
		Value: amount,
	}
	stdOutLogger.Println("Creating invoice for a new API request")
	res, err := c.lndClient.AddInvoice(c.ctx, &invoice)
	if err != nil {
		return result, err
	}

	result.ImplDepID = hex.EncodeToString(res.RHash)
	result.PaymentHash = result.ImplDepID
	result.PaymentRequest = res.PaymentRequest
	return result, nil
}

// CheckInvoice takes an invoice ID (LN node implementation specific) and checks if the corresponding invoice was settled.
// An error is returned if no corresponding invoice was found.
// False is returned if the invoice isn't settled.
func (c LNDclient) CheckInvoice(id string) (bool, error) {
	// In the case of lnd, the ID is the hex encoded preimage hash.
	plainHash, err := hex.DecodeString(id)
	if err != nil {
		return false, err
	}

	stdOutLogger.Printf("Checking invoice for hash %v\n", id)

	// Get the invoice for that hash
	paymentHash := lnrpc.PaymentHash{
		RHash: plainHash,
		// Hex encoded, must be exactly 32 byte
		RHashStr: id,
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

// Pay pays the invoice and returns the preimage (hex encoded) on success, or an error on failure.
func (c LNDclient) Pay(invoice string) (string, error) {
	// Decode payment request (a.k.a. invoice).
	// TODO: Decoded values are only used for logging, so maybe make this optional to make fewer RPC calls
	payReqString := lnrpc.PayReqString{
		PayReq: invoice,
	}
	decodedPayReq, err := c.lndClient.DecodePayReq(c.ctx, &payReqString)
	if err != nil {
		return "", err
	}

	// Send payment
	sendReq := lnrpc.SendRequest{
		PaymentRequest: invoice,
	}
	stdOutLogger.Printf("Sending payment with %v Satoshis to %v (memo: \"%v\")",
		decodedPayReq.NumSatoshis, decodedPayReq.Destination, decodedPayReq.Description)
	sendRes, err := c.lndClient.SendPaymentSync(c.ctx, &sendReq)
	if err != nil {
		return "", err
	}
	// Even if err is nil, this just means the RPC call was successful, not the payment was successful
	if sendRes.PaymentError != "" {
		return "", errors.New(sendRes.PaymentError)
	}

	hexPreimage := hex.EncodeToString(sendRes.PaymentPreimage)
	return string(hexPreimage), nil
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
	// Path to the macaroon file that your LND node uses.
	// "invoice.macaroon" if you only use the GenerateInvoice() and CheckInvoice() methods
	// (required by the middleware in the package "wall").
	// "admin.macaroon" if you use the Pay() method (required by the client in the package "pay").
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

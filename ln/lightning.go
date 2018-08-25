package ln

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
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

// NewLNDclient creates a new LNDclient instance.
func NewLNDclient(address string, certFile string, macaroonFile string) (LNDclient, error) {
	result := LNDclient{}

	// Set up a connection to the server.
	creds, err := credentials.NewClientTLSFromFile(certFile, "")
	if err != nil {
		return result, err
	}
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return result, err
	}
	c := lnrpc.NewLightningClient(conn)

	// Add the macaroon to the outgoing context

	macaroon, err := ioutil.ReadFile(macaroonFile)
	if err != nil {
		return result, err
	}
	// Value must be the hex representation of the file content
	macaroonHex := fmt.Sprintf("%X", string(macaroon))
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "macaroon", macaroonHex)

	result = LNDclient{
		conn:      conn,
		ctx:       ctx,
		lndClient: c,
	}

	return result, nil
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

// CheckInvoice takes a Base64 encoded preimage, fetches the corresponding invoice,
// and checks if the invoice was settled.
// An error is returned if no corresponding invoice was found.
// False is returned if the invoice isn't settled.
func (c LNDclient) CheckInvoice(preimage string) (bool, error) {
	// Hash the preimage so we can get the invoice that belongs to it to check if it's settled
	decodedPreimage, err := base64.StdEncoding.DecodeString(preimage)
	if err != nil {
		return false, err
	}
	hash := sha256.Sum256([]byte(decodedPreimage))
	hashSlice := hash[:]

	// Get the invoice for that hash
	paymentHash := lnrpc.PaymentHash{
		RHash: hashSlice,
		// Hex encoded, must be exactly 32 byte
		RHashStr: hex.EncodeToString(hashSlice),
	}
	encodedHash := base64.StdEncoding.EncodeToString(hashSlice)
	log.Printf("Checking invoice for hash %v\n", encodedHash)
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

// HashPreimage hashes the Base64 preimage and encodes the hash in Base64.
// It's the same format that's being shown by lncli listinvoices (preimage as well as hash).
func HashPreimage(preimage string) (string, error) {
	decodedPreimage, err := base64.StdEncoding.DecodeString(preimage)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(decodedPreimage))
	hashSlice := hash[:]
	encodedHash := base64.StdEncoding.EncodeToString(hashSlice)
	return encodedHash, nil
}

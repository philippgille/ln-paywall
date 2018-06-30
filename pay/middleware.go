package pay

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/philippgille/go-paypercall/lnrpc"
)

// NewMiddleware returns a function which you can use within an http.HandlerFunc chain.
// The amount parameter is the amount of satoshis you want to have paid for one API call.
// The address parameter is the address of your LND node, including the port.
// The certFile parameter is the path to the "tls.cert" file that your LND node uses.
// The macaroonFile parameter is the path to the "admin.macaroon" file that your LND node uses.
func NewMiddleware(amount int64, address string, certFile string, macaroonFile string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Check if the request contains a header with the preimage that we need to check if the requester paid
			preimage := r.Header.Get("x-preimage")
			if preimage == "" {
				// Generate the invoice
				invoice, err := generateInvoice(amount, address, certFile, macaroonFile)
				if err != nil {
					errorMsg := fmt.Sprintf("Couldn't generate invoice: %+v", err)
					log.Println(errorMsg)
					http.Error(w, errorMsg, http.StatusBadRequest)
				} else {
					// Note: w.Header().Set(...) must be called before w.WriteHeader(...)!
					w.Header().Set("Content-Type", "application/vnd.lightning.bolt11")
					w.WriteHeader(http.StatusPaymentRequired)
					// The actual invoice goes into the body
					w.Write([]byte(invoice))
				}
			} else {
				// Check if the provided preimage belongs to a settled API payment invoice and that it wasn't already used
				ok, err := checkPreimage(preimage, address, certFile, macaroonFile)
				if err != nil {
					errorMsg := fmt.Sprintf("An error occured during checking the preimage: %+v", err)
					log.Printf("%v\n", errorMsg)
					http.Error(w, errorMsg, http.StatusBadRequest)
				} else {
					if !ok {
						log.Printf("The provided preimage is invalid: %v\n", preimage)
						http.Error(w, "The provided preimage is invalid", http.StatusBadRequest)
					} else {
						next.ServeHTTP(w, r)
					}
				}
			}
		}
	}
}

func generateInvoice(amount int64, address string, certFile string, macaroonFile string) (string, error) {
	// Create the client
	c, ctx, conn, err := newLightningClient(address, certFile, macaroonFile)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Create the request and send it
	invoice := lnrpc.Invoice{
		Memo:  "API call payment",
		Value: amount,
	}
	log.Println("Creating invoice for a new API request")
	res, err := c.AddInvoice(ctx, &invoice)
	if err != nil {
		return "", err
	}

	return res.GetPaymentRequest(), nil
}

// checkPreimage takes a Base64 encoded preimage and checks if it's a valid preimage for an API payment.
func checkPreimage(preimage string, address string, certFile string, macaroonFile string) (bool, error) {
	// Hash the preimage so we can get the invoice that belongs to it to check if it's settled

	decodedPreimage, err := base64.StdEncoding.DecodeString(preimage)
	if err != nil {
		return false, err
	}
	hash := sha256.Sum256([]byte(decodedPreimage))
	hashSlice := hash[:]

	// Get the invoice for that hash

	// Create the client
	c, ctx, conn, err := newLightningClient(address, certFile, macaroonFile)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	// Create the request and send it
	paymentHash := lnrpc.PaymentHash{
		RHash: hashSlice,
		// Hex encoded, must be exactly 32 byte
		RHashStr: hex.EncodeToString(hashSlice),
	}
	encodedHash := base64.StdEncoding.EncodeToString(hashSlice)
	log.Printf("Checking invoice for hash %v\n", encodedHash)
	invoice, err := c.LookupInvoice(ctx, &paymentHash)
	if err != nil {
		return false, err
	}

	// Perform checks on the invoice

	// Check if invoice was settled
	if !invoice.GetSettled() {
		return false, nil
	}

	// Check if it was already used before
	// TODO: implement

	return true, nil
}

func newLightningClient(address string, certFile string, macaroonFile string) (lnrpc.LightningClient, context.Context, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	creds, err := credentials.NewClientTLSFromFile(certFile, "")
	if err != nil {
		return nil, nil, nil, err
	}
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, nil, nil, err
	}
	c := lnrpc.NewLightningClient(conn)

	// Add the macaroon to the outgoing context

	macaroon, err := ioutil.ReadFile(macaroonFile)
	if err != nil {
		return nil, nil, nil, err
	}
	// Value must be the hex representation of the file content
	macaroonHex := fmt.Sprintf("%X", string(macaroon))
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "macaroon", macaroonHex)

	return c, ctx, conn, nil
}

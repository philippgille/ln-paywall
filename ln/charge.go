package ln

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ChargeClient is an implementation of the wall.LNclient interface for "Lightning Charge"
// running on top of the c-lightning Lightning Network node implementation.
type ChargeClient struct {
	client   *http.Client
	baseURL  string
	apiToken string
}

// GenerateInvoice generates an invoice with the given price and memo.
func (c ChargeClient) GenerateInvoice(amount int64, memo string) (string, error) {
	data := make(url.Values)
	// Possible values as documented in https://github.com/ElementsProject/lightning-charge/blob/master/README.md:
	// msatoshi, currency, amount, description, expiry, metadata and webhook
	// But with *either* msatoshi *or* currency + amount
	mSatoshi := strconv.FormatInt(1000*amount, 10)
	data.Add("msatoshi", mSatoshi)
	data.Add("description", memo)

	// Send request
	req, err := http.NewRequest("POST", c.baseURL+"/invoice", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("api-token", c.apiToken) // This might seem strange, but it's how Lightning Charge expects it
	res, err := c.client.Do(req)
	if err != nil {
		return "", err
	}

	// Read and deserialize response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	err = res.Body.Close()
	if err != nil {
		return "", err
	}
	invoice, err := deserializeInvoice(body)
	if err != nil {
		return "", err
	}

	return invoice.Payreq, nil
}

// CheckInvoice takes a hex encoded preimage and checks if the corresponding invoice was settled.
// This is done by fetching ALL invoices (for reasons outlined below) and looking for a matching preimage hash,
// then checking if the found invoice was settled.
// An error is returned if the preimage isn't properly encoded or if no corresponding invoice was found.
// False is returned if the invoice isn't settled.
//
// Implementation notes: Lightning Charge doesn't allow to fetch an invoice via a preimage or preimage hash,
// which is fine by itself, but doesn't fit our implementation, which was focused on lnd at first
// (which allows to fetch invoices via preimage hash).
// In the future, for multiple reasons (for example GitHub issue #16),
// the client won't send the preimage anymore but just a token,
// similar to or the same as with ElementProject's paypercall.
func (c ChargeClient) CheckInvoice(preimageHex string) (bool, error) {
	preimageHashHex, err := HashPreimage(preimageHex)
	if err != nil {
		return false, err
	}

	log.Printf("Checking invoice for hash %v\n", preimageHashHex)

	// Fetch all existing invoices
	req, err := http.NewRequest("GET", c.baseURL+"/invoices", nil)
	if err != nil {
		return false, err
	}
	req.SetBasicAuth("api-token", c.apiToken) // This might seem strange, but it's how Lightning Charge expects it
	res, err := c.client.Do(req)
	if err != nil {
		return false, err
	}

	invoicesJSON, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return false, err
	}
	err = res.Body.Close()
	if err != nil {
		return false, err
	}

	invoices, err := deserializeInvoices(invoicesJSON)
	if err != nil {
		return false, err
	}

	// Iterate through them and find the one which the request's preimage is for
	var foundInvoice chargeInvoice
	for _, invoice := range invoices {
		if invoice.Rhash == preimageHashHex {
			foundInvoice = invoice
		}
	}
	if foundInvoice.ID == "" {
		// For this error the string "unable to locate invoice" MUST be used.
		// This is a workaround until wall.handlePreimage() doesn't rely on that error string anymore.
		// TODO: Improve
		return false, errors.New("unable to locate invoice")
	}

	// An invoice was found, now check if it was settled
	if foundInvoice.Status == "unpaid" {
		return false, nil
	} else if foundInvoice.Status == "paid" {
		// All checks for errors are done, return ok
		return true, nil
	} else {
		// Unknown status
		// TODO: Find out which statuses exist and handle them properly
		return false, errors.New("The invoice found in Lightning Charge has an unknown / unhandled status")
	}
}

// NewChargeClient creates a new ChargeClient instance.
func NewChargeClient(chargeOptions ChargeOptions) (ChargeClient, error) {
	result := ChargeClient{}

	chargeOptions = assignChargeDefaultValues(chargeOptions)

	result.client = http.DefaultClient
	// Make sure the address doesn't end with "/", so that in the other functions
	// we can rely on that it's ok to add for example "/invoice" to the baseURL.
	result.baseURL = strings.TrimSuffix(chargeOptions.Address, "/")
	result.apiToken = chargeOptions.APItoken

	return result, nil
}

// ChargeOptions are the options for the connection to Lightning Charge.
type ChargeOptions struct {
	// Address of your Lightning Charge server, including the protocol (e.g. "https://") and port.
	// Optional ("http://localhost:9112" by default).
	Address string
	// APItoken for authenticating the request to Lightning Charge.
	// The token is configured when Lightning Charge is started.
	APItoken string
}

// DefaultChargeOptions provides default values for ChargeOptions.
var DefaultChargeOptions = ChargeOptions{
	Address: "http://localhost:9112",
}

func assignChargeDefaultValues(chargeOptions ChargeOptions) ChargeOptions {
	if chargeOptions.Address == "" {
		chargeOptions.Address = DefaultChargeOptions.Address
	}

	return chargeOptions
}

// chargeInvoice is the Go data structure for the invoice JSON from Lightning Charge.
//
// Example JSON:
// {
//   "id": "4ya51ILHmKU6C8UjfEcnR",
//   "msatoshi": "1000000",
//   "description": "Lightning Charge Invoice",
//   "rhash":
//     "4bb443f021f20b9f57fd12c6c10fdf88252c0a1ae6fdeccfa3210b7b8190c464",
//   "payreq":
//     "lntb10u1pdegzmfpp5fw6y8upp7g9e74laztrvzr7l3qjjczs6um77enaryy9hhqvsc3jqdp8f35kw6r5de5kueeqgd5xzun8v5syjmnkda5kxegcqp2vagpefsetakat4erfvs2qk7nx28wa5nwa4ld0ewmhl99x6g8kj737xhpfwctx2vs0qqx38yu7td7u9rta4mx0xrdk4kstp29zvwmfhgp24rrx9",
//   "expires_at": 1536432505,
//   "created_at": 1536428905,
//   "metadata": null,
//   "status": "unpaid"
// }
//
// Automatically converted via https://transform.now.sh/json-to-go/.
type chargeInvoice struct {
	ID          string      `json:"id"`
	Msatoshi    string      `json:"msatoshi"`
	Description string      `json:"description"`
	Rhash       string      `json:"rhash"`
	Payreq      string      `json:"payreq"`
	ExpiresAt   int         `json:"expires_at"`
	CreatedAt   int         `json:"created_at"`
	Metadata    interface{} `json:"metadata"`
	Status      string      `json:"status"`
}

// deserializeInvoice converts an invoice JSON object to an instance of the chargeInvoice struct
func deserializeInvoice(invoiceJSON []byte) (chargeInvoice, error) {
	result := chargeInvoice{}
	err := json.Unmarshal(invoiceJSON, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// deserializeInvoices converts an invoice JSON array of multiple invoices to a slice of instances of the chargeInvoice struct
func deserializeInvoices(invoicesJSON []byte) ([]chargeInvoice, error) {
	invoiceCountHeuristic := len(invoicesJSON) / 600 // Assuming one byte equals one character, and 600 characters per invoice
	result := make([]chargeInvoice, invoiceCountHeuristic)
	err := json.Unmarshal(invoicesJSON, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

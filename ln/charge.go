package ln

import (
	"encoding/json"
	"errors"
	"io/ioutil"
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
func (c ChargeClient) GenerateInvoice(amount int64, memo string) (Invoice, error) {
	result := Invoice{}

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
		return result, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("api-token", c.apiToken) // This might seem strange, but it's how Lightning Charge expects it
	stdOutLogger.Println("Creating invoice for a new API request")
	res, err := c.client.Do(req)
	if err != nil {
		return result, err
	}

	// Read and deserialize response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return result, err
	}
	err = res.Body.Close()
	if err != nil {
		return result, err
	}
	invoice, err := deserializeInvoice(body)
	if err != nil {
		return result, err
	}

	result.ImplDepID = invoice.ID
	result.PaymentHash = invoice.Rhash
	result.PaymentRequest = invoice.Payreq
	return result, nil
}

// CheckInvoice takes an invoice ID (LN node implementation specific) and checks if the corresponding invoice was settled.
// An error is returned if the invoice info couldn't be fetched from Lightning Charge or deserialized etc.
// False is returned if the invoice isn't settled.
func (c ChargeClient) CheckInvoice(id string) (bool, error) {
	stdOutLogger.Printf("Checking invoice %v\n", id)

	// Fetch invoice
	req, err := http.NewRequest("GET", c.baseURL+"/invoice/"+id, nil)
	if err != nil {
		return false, err
	}
	req.SetBasicAuth("api-token", c.apiToken) // This might seem strange, but it's how Lightning Charge expects it
	res, err := c.client.Do(req)
	if err != nil {
		return false, err
	}

	invoiceJSON, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return false, err
	}
	err = res.Body.Close()
	if err != nil {
		return false, err
	}

	invoice, err := deserializeInvoice(invoiceJSON)
	if err != nil {
		return false, err
	}

	if invoice.Status == "unpaid" {
		return false, nil
	} else if invoice.Status == "paid" {
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

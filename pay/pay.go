package pay

import (
	"errors"
	"io/ioutil"
	"net/http"
)

// LNclient is the abstraction of a Lightning Network node client for paying LN invoices.
type LNclient interface {
	// Pay pays the invoice and returns the preimage on success, or an error on failure.
	Pay(invoice string) (string, error)
}

// Client is an HTTP client, which handles "Payment Required" interruptions transparently.
// It must be initially set up with a connection the Lightning Network node that should handle the payments
// and from then on it's meant to be used as an alternative to the "net/http.Client".
// The calling code only needs to call the Do(...) method once, instead of handling
// "402 Payment Required" responses and re-sending the original request after payment.
type Client struct {
	c *http.Client
	l LNclient
}

// Get sends an HTTP GET request to the given URL and automatically handles the required payment in the background.
// It does this by sending its own request to the URL + path of the given request
// to trigger a "402 Payment Required" response with an invoice.
// It then pays the invoice via the configured Lightning Network node.
// Finally it sends the originally intended (given) request with an additional HTTP header and returns the response.
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

// Do sends the given request and automatically handles the required payment in the background.
// It does this by sending its own request to the URL + path of the given request
// to trigger a "402 Payment Required" response with an invoice.
// It then pays the invoice via the configured Lightning Network node.
// Finally it sends the originally intended (given) request with an additional HTTP header and returns the response.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// Send first request, no data (query params or body) required

	invoiceReq, err := http.NewRequest(req.Method, req.URL.Scheme+"://"+req.URL.Host+req.URL.EscapedPath(), nil)
	if err != nil {
		return nil, err
	}

	res, err := c.c.Do(invoiceReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Expect "402 Payment Required" status code in response
	if res.StatusCode != http.StatusPaymentRequired {
		return nil, errors.New("Request expected to trigger \"402 Payment Required\" response, but was: " + " " + res.Status)
	}

	invoice, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// Pay invoice
	hexPreimage, err := c.l.Pay(string(invoice))
	if err != nil {
		return nil, err
	}

	// Add preimage to the original request's headers
	req.Header.Add("X-Preimage", hexPreimage)

	// Send original request and return response or error
	return c.c.Do(req)
}

// NewClient creates a new pay.Client instance.
// You can pass nil as httpClient, in which case the http.DefaultClient will be used.
func NewClient(httpClient *http.Client, lnClient LNclient) Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return Client{
		c: httpClient,
		l: lnClient,
	}
}

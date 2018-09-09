This Docker image contains an example API for the project "ln-paywall". For more info about the project visit https://github.com/philippgille/ln-paywall.

Prerequisites
-------------

- A running lnd node, either on a remote host and accessible from outside, or on the same host, in which case you can either start this container in "host network" mode, or use the container's gateway IP address to reach the host's localhost

Usage
-----

1. Create a data directory on the host: `mkdir data`
2. Copy the `tls.cert` and `invoice.macaroon` from your lnd to the `data/` directory
3. Run the container: `docker run -d --name qr-code -v $(pwd)/data/:/root/data/ -p 8080:8080 philippgille/qr-code -addr "123.123.123.123:10009"`
4. Send a request: `curl http://localhost:8080/qr?data=testtext`
5. Pay the invoice from the response via the Lightning Network
6. Send the request again, this time with the preimage as payment proof (hex encoded) and the data as query parameters: `curl -H "x-preimage: 119969c2338798cd56708126b5d6c0f6f5e75ed38da7a409b0081d94b4dacbf8" http://localhost:8080/qr?data=testtext`

The response contains the QR code as PNG image.

Options
-------

```
  -addr string
        Address of the lnd node (including gRPC port) (default "localhost:10009")
  -dataDir string
        Relative path to the data directory, where tls.cert and invoice.macaroon are located (default "data/")
  -price int
        Price of one request in Satoshis (at an exchange rate of $1,000 for 1 BTC 1000 Satoshis would be $0.01) (default 1000)
```

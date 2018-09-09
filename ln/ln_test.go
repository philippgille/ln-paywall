package ln_test

import (
	"testing"

	"github.com/philippgille/ln-paywall/ln"
)

// TestHashPreimage tests if the result of the HashPreimage function is correct.
func TestHashPreimage(t *testing.T) {
	// Correct preimage form, taken from a payment JSON in lnd
	preimageHex := "119969c2338798cd56708126b5d6c0f6f5e75ed38da7a409b0081d94b4dacbf8"
	// Taken from the same payment JSON in lnd
	expected := "bf3e0e73d4bb1ee9d68ca8d1078213d059e23d6e1c8a14b3df93faf87aa4fed3"
	actual, err := ln.HashPreimage(preimageHex)
	if err != nil {
		t.Errorf("An error occurred during the test: %v\n", err)
	}
	if actual != expected {
		t.Errorf("Expected %v, but was %v\n", expected, actual)
	}
}

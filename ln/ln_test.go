package ln_test

import (
	"testing"

	"github.com/philippgille/ln-paywall/ln"
)

// TestHashPreimage tests if the result of the HashPreimage function is correct.
func TestHashPreimage(t *testing.T) {
	// Correct preimage form, taken from an invoice JSON in lnd
	preimage := "CtkknKfJ237o8jbippg5NziFZgkUvk9cN20NCuPvqxQ="
	// Taken from the same invoice JSON in lnd
	expected := "iYYYRTzq0XZgBo1aIAv4KZTzq1PCTmf6nnWvyHMB4C0="
	actual, err := ln.HashPreimage(preimage)
	if err != nil {
		t.Errorf("An error occurred during the test: %v\n", err)
	}
	if actual != expected {
		t.Errorf("Expected %v, but was %v\n", expected, actual)
	}
}

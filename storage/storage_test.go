package storage_test

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/philippgille/ln-paywall/wall"
)

// testStorageClient tests if reading and writing to the storage works properly.
func testStorageClient(storageClient wall.StorageClient, t *testing.T) {
	key := strconv.FormatInt(rand.Int63(), 10)

	// Initially the key shouldn't exist, thus "was used" should be false
	expected := false
	actual, err := storageClient.WasUsed(key)
	if err != nil {
		t.Error(err)
	}
	if actual != expected {
		t.Errorf("Expected: %v, but was: %v", expected, actual)
	}

	// Set the "key" to be used
	err = storageClient.SetUsed(key)
	if err != nil {
		t.Error(err)
	}

	// Check usage again, this time "was used" should be true
	expected = true
	actual, err = storageClient.WasUsed(key)
	if err != nil {
		t.Error(err)
	}
	if actual != expected {
		t.Errorf("Expected: %v, but was: %v", expected, actual)
	}
}

// interactWithStorage reads from and writes to the DB. Meant to be executed in a goroutine.
// Does NOT check if the DB works correctly (that's done by another test), only checks for errors
func interactWithStorage(storageClient wall.StorageClient, key string, t *testing.T, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	// Read
	_, err := storageClient.WasUsed(key)
	if err != nil {
		t.Error(err)
	}
	// Write
	err = storageClient.SetUsed(key)
	if err != nil {
		t.Error(err)
	}
	// Read
	_, err = storageClient.WasUsed(key)
	if err != nil {
		t.Error(err)
	}
}

package storage_test

import (
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/philippgille/ln-paywall/ln"
	"github.com/philippgille/ln-paywall/storage"
	"github.com/philippgille/ln-paywall/wall"
)

// TestBoltClientImpl tests if the BoltClient struct implements the StorageClient interface.
// This doesn't happen at runtime, but at compile time.
func TestBoltClientImpl(t *testing.T) {
	t.SkipNow()
	invoiceOptions := wall.InvoiceOptions{}
	lnClient := ln.LNDclient{}
	boltClient, _ := storage.NewBoltClient(storage.DefaultBoltOptions)
	wall.NewHandlerFuncMiddleware(invoiceOptions, lnClient, boltClient)
	wall.NewHandlerMiddleware(invoiceOptions, lnClient, boltClient)
	wall.NewGinMiddleware(invoiceOptions, lnClient, boltClient)
	wall.NewEchoMiddleware(invoiceOptions, lnClient, boltClient, nil)
}

// TestBoltClient tests if reading and writing to the storage works properly.
func TestBoltClient(t *testing.T) {
	boltOptions := storage.BoltOptions{
		Path: generateRandomTempDbPath(),
	}
	boltClient, err := storage.NewBoltClient(boltOptions)
	if err != nil {
		t.Error(err)
	}

	testStorageClient(boltClient, t)
}

// TestBoltClientConcurrent launches a bunch of goroutines that concurrently work with one BoltClient.
// The BoltClient works with a single file, so everything should be locked properly.
// The locking is implemented in the bbolt package, but test it nonetheless.
func TestBoltClientConcurrent(t *testing.T) {
	boltOptions := storage.BoltOptions{
		Path: generateRandomTempDbPath(),
	}
	boltClient, err := storage.NewBoltClient(boltOptions)
	if err != nil {
		t.Error(err)
	}

	goroutineCount := 1000

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(goroutineCount) // Must be called before any goroutine is started
	for i := 0; i < goroutineCount; i++ {
		go interactWithStorage(boltClient, strconv.Itoa(i), t, &waitGroup)
	}
	waitGroup.Wait()

	// Now make sure that all values are in the storage
	expected := true
	for i := 0; i < goroutineCount; i++ {
		actual, _ := boltClient.WasUsed(strconv.Itoa(i))
		if actual != expected {
			t.Errorf("Expected: %v, but was: %v", expected, actual)
		}
	}
}

func generateRandomTempDbPath() string {
	return os.TempDir() + "/" + strconv.FormatInt(rand.Int63(), 10) + ".db"
}

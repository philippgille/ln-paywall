package storage

import (
	"sync"

	bolt "github.com/coreos/bbolt"
)

var bucketName = "ln-paywall"

// BoltClient is a StorageClient implementation for bbolt (formerly known as Bolt / Bolt DB).
type BoltClient struct {
	db   *bolt.DB
	lock *sync.Mutex
}

// WasUsed checks if the preimage was used for a previous payment already.
func (c BoltClient) WasUsed(preimage string) (bool, error) {
	var result bool
	err := c.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		v := b.Get([]byte(preimage))
		if v != nil {
			result = true
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return result, nil
}

// SetUsed stores the information that a preimage has been used for a payment.
func (c BoltClient) SetUsed(preimage string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	err := c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		err := b.Put([]byte(preimage), []byte("1"))
		return err
	})
	if err != nil {
		return err
	}
	return nil
}

// BoltOptions are the options for the BoltClient.
type BoltOptions struct {
	// Path of the DB file.
	// Optional ("ln-paywall.db" by default).
	Path string
}

// DefaultBoltOptions is a BoltOptions object with default values.
// Path: "ln-paywall.db"
var DefaultBoltOptions = BoltOptions{
	Path: "ln-paywall.db",
}

// NewBoltClient creates a new BoltClient.
// Note: Bolt uses an exclusive write lock on the database file so it cannot be shared by multiple processes.
// This shouldn't be a problem when you use one file for one middleware, like this:
//  // ...
//  boltClient, err := wall.NewBoltClient(wall.DefaultBoltOptions) // Uses file "ln-paywall.db"
//  if err != nil {
//      panic(err)
//  }
//  defer boltClient.Close()
//  r.Use(wall.NewGinMiddleware(invoiceOptions, lndOptions, boltClient))
//  // ...
// Also don't worry about closing the Bolt DB, the middleware opens it once and uses it for the duration of its lifetime.
// When the web service is stopped, the DB file lock is released automatically.
func NewBoltClient(boltOptions BoltOptions) (BoltClient, error) {
	result := BoltClient{}

	// Set default values
	if boltOptions.Path == "" {
		boltOptions.Path = DefaultBoltOptions.Path
	}

	// Open DB
	db, err := bolt.Open(boltOptions.Path, 0600, nil)
	if err != nil {
		return result, err
	}

	// Create a bucket if it doesn't exist yet.
	// In Bolt key/value pairs are stored to and read from buckets.
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return result, err
	}

	result = BoltClient{
		db:   db,
		lock: &sync.Mutex{},
	}

	return result, nil
}

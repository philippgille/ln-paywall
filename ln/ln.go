package ln

import (
	"crypto/sha256"
	"encoding/base64"
)

// HashPreimage hashes the Base64 preimage and encodes the hash in Base64.
// It's the same format that's being shown by lncli listinvoices (preimage as well as hash).
func HashPreimage(preimage string) (string, error) {
	decodedPreimage, err := base64.StdEncoding.DecodeString(preimage)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(decodedPreimage))
	hashSlice := hash[:]
	encodedHash := base64.StdEncoding.EncodeToString(hashSlice)
	return encodedHash, nil
}

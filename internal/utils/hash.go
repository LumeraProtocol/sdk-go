package utils

import (
	"lukechampine.com/blake3"
)

// HashFile computes the Blake3 hash of a file
func HashFile(data []byte) []byte {
	hasher := blake3.New(32, nil)
	_, err := hasher.Write(data)
	if err != nil {
		return nil
	}
	return hasher.Sum(nil)
}

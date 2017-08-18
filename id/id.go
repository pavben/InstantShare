// Package id contains facilities for generating and validating random IDs
// which are meant to be used as uploaded filenames.
package id

import (
	"crypto/rand"
	"math"
	"math/big"
)

// Generate a random ID like "1njfizqgeukrq", "3l44ze7pf47fd", "35dgd4n5ryup", etc.
func Generate() (id string, err error) {
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return n.Text(36), nil
}

var max = new(big.Int).SetUint64(math.MaxUint64)

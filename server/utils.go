package main

import (
	"math/rand"
	"strconv"
)

func GenerateRandomString() string {
	randomNumber := uint64(rand.Int63())

	return strconv.FormatUint(randomNumber, 36)
}

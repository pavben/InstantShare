package main

import (
	"math/rand"
	"strconv"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateRandomString() string {
	randomNumber := uint64(rand.Int63())

	return strconv.FormatUint(randomNumber, 36)
}

package main

import (
	"math/rand"
	"strconv"
)

type FileId string

func GenerateNewFileID() FileId {
	randomNumber := uint64(rand.Int63())

	return FileId(strconv.FormatUint(randomNumber, 36))
}

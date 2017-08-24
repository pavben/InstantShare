package id_test

import (
	"testing"

	"github.com/pavben/InstantShare/id"
)

func TestGenerate(t *testing.T) {
	for i := 0; i < 100; i++ {
		_, err := id.Generate()
		if err != nil {
			t.Error(err)
		}
	}
}

package main

import (
	"io"
	"time"
)

type fileStore interface {
	GetFileReader(fileName string) (fileReader, error)
	GetFileWriter(fileName string) (io.WriteCloser, error)
	RemoveFile(fileName string) error
}

type fileReader interface {
	ContentType() string
	Size() (int, error)
	io.ReadSeeker
	io.Closer

	// HACK: Should use Stat() method; return error.
	ModTime() time.Time
}

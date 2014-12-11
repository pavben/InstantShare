package main

import (
	"io"
	"time"
)

type FileStore interface {
	GetFileReader(fileName string) (FileReader, error)
	GetFileWriter(fileName string) (FileWriter, error)
	RemoveFile(fileName string) error
}

type FileReader interface {
	ContentType() string
	Size() (int, error)
	io.ReadSeeker
	io.Closer

	// HACK: Should use Stat() method; return error.
	ModTime() time.Time
}

type FileWriter interface {
	Write(p []byte) (int, error)
	Close() error
}

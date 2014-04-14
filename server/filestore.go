package main

type FileStore interface {
	GetFileReader(fileName string) FileReader
	GetFileWriter(fileName string) FileWriter
}

type FileReader interface {
	ContentType() string
	Size() (int, error)
	Read(p []byte) (int, error)
	Close() error
}

type FileWriter interface {
	Write(p []byte) (int, error)
	Close() error
}

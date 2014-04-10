package main

type FileStore interface {
	GetFile(name string) File
}

type File interface {
	Size() (int, error)
	Read(p []byte) (int, error)
	Close() error
}

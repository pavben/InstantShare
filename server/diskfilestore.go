package main

import (
	"errors"
	"math"
	"os"
)

var (
	ErrFileTooBig = errors.New("File too big")
)

const basePath = "files/"

type DiskFileStore struct {
}

func NewDiskFileStore() (FileStore, error) {
	fileStore := &DiskFileStore{}

	_, err := os.Stat(basePath)

	// if failed to stat, create the dir
	if err != nil {
		err = os.Mkdir(basePath, 0700)
	}

	// if failed to stat and failed to create dir, fail
	if err != nil {
		return nil, err
	}

	return fileStore, nil
}

func (self *DiskFileStore) GetFileReader(fileName string) FileReader {
	file, err := os.Open(FileNameToPath(fileName))

	if err != nil {
		return nil
	}

	return &DiskFileReader{
		file:        file,
		contentType: ContentTypeFromFileName(fileName),
	}
}

func (self *DiskFileStore) GetFileWriter(fileName string) FileWriter {
	file, err := os.Create(FileNameToPath(fileName))

	if err != nil {
		return nil
	}

	return &DiskFileWriter{
		file: file,
	}
}

type DiskFileReader struct {
	file        *os.File
	contentType string
}

func (self *DiskFileReader) ContentType() string {
	return self.contentType
}

func (self *DiskFileReader) Size() (int, error) {
	fileInfo, err := self.file.Stat()

	if err != nil {
		return -1, err
	}

	fileSizeInt64 := fileInfo.Size()

	if fileSizeInt64 > math.MaxInt32 {
		return -1, ErrFileTooBig
	}

	return int(fileSizeInt64), nil
}

func (self *DiskFileReader) Read(p []byte) (int, error) {
	return self.file.Read(p)
}

func (self *DiskFileReader) Close() (err error) {
	err = self.file.Close()

	if err == nil {
		self.file = nil
	}

	return
}

type DiskFileWriter struct {
	file *os.File
}

func (self *DiskFileWriter) Write(p []byte) (int, error) {
	bytesWritten, err := self.file.Write(p)

	if err != nil {
		return bytesWritten, err
	}

	// the data written must be available for reading immediately as Write returns
	err = self.file.Sync()

	return bytesWritten, err
}

func (self *DiskFileWriter) Close() (err error) {
	err = self.file.Close()

	if err == nil {
		self.file = nil
	}

	return
}

// TODO: this is to be hidden from outside
func FileNameToPath(fileName string) string {
	return basePath + fileName
}

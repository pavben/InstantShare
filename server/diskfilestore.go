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

func (self *DiskFileStore) GetFile(fileName string) File {
	file, err := os.Open(FileNameToPath(fileName))

	if err != nil {
		return nil
	}

	return &DiskFile{
		file: file,
	}
}

type DiskFile struct {
	file *os.File
}

func (self *DiskFile) Size() (int, error) {
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

func (self *DiskFile) Read(p []byte) (int, error) {
	return self.file.Read(p)
}

func (self *DiskFile) Close() (err error) {
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

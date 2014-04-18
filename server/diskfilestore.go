package main

import (
	"errors"
	"math"
	"os"
	"path/filepath"
)

var (
	ErrFileTooBig                   = errors.New("File too big")
	ErrDiskFileStoreDirDoesNotExist = errors.New("The specified path for disk file store is not an existing directory")
)

type DiskFileStore struct {
	basePath string
}

func NewDiskFileStore(fileStorePath string) (FileStore, error) {
	// It it's not an existing directory, abort and ask the user to create it or specify existing directory.
	s, err := os.Stat(fileStorePath)
	isDirectory := err == nil && s.IsDir()
	if !isDirectory {
		return nil, ErrDiskFileStoreDirDoesNotExist
	}

	fileStore := &DiskFileStore{basePath: fileStorePath}

	return fileStore, nil
}

func (self *DiskFileStore) GetFileReader(fileName string) (FileReader, error) {
	file, err := os.Open(self.fileNameToPath(fileName))

	if err != nil {
		return nil, err
	}

	diskFileReader := &DiskFileReader{
		file:        file,
		contentType: ContentTypeFromFileName(fileName),
	}

	return diskFileReader, nil
}

func (self *DiskFileStore) GetFileWriter(fileName string) (FileWriter, error) {
	file, err := os.Create(self.fileNameToPath(fileName))

	if err != nil {
		return nil, err
	}

	diskFileWriter := &DiskFileWriter{
		file: file,
	}

	return diskFileWriter, nil
}

func (self *DiskFileStore) RemoveFile(fileName string) error {
	return os.Remove(self.fileNameToPath(fileName))
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

func (self *DiskFileStore) fileNameToPath(fileName string) string {
	return filepath.Join(self.basePath, fileName)
}

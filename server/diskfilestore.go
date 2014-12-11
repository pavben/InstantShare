package main

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"time"
)

var (
	ErrFileTooBig = errors.New("File too big")
)

const basePath = "files"

type DiskFileStore struct{}

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

func (self *DiskFileStore) fileNameToPath(fileName string) string {
	return filepath.Join(basePath, fileName)
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

func (self *DiskFileReader) ModTime() time.Time {
	fi, err := self.file.Stat()
	if err != nil {
		return time.Time{}
	}

	return fi.ModTime()
}

func (self *DiskFileReader) Read(p []byte) (int, error) {
	/*n, err := self.file.Read(p)
	fmt.Println("Read (len, n, err):", len(p), n, err)
	return n, err*/
	return self.file.Read(p)
}

func (self *DiskFileReader) Seek(offset int64, whence int) (int64, error) {
	return self.file.Seek(offset, whence)
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

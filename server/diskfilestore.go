package main

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"time"
)

var (
	errFileTooBig = errors.New("File too big")
)

const basePath = "files"

type diskFileStore struct{}

func newDiskFileStore() (fileStore, error) {
	fileStore := &diskFileStore{}

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

func (diskFileStore *diskFileStore) GetFileReader(fileName string) (fileReader, error) {
	file, err := os.Open(diskFileStore.fileNameToPath(fileName))
	if err != nil {
		return nil, err
	}

	diskFileReader := &diskFileReader{
		file:        file,
		contentType: contentTypeFromFileName(fileName),
	}

	return diskFileReader, nil
}

func (diskFileStore *diskFileStore) GetFileWriter(fileName string) (fileWriter, error) {
	file, err := os.Create(diskFileStore.fileNameToPath(fileName))
	if err != nil {
		return nil, err
	}

	diskFileWriter := &diskFileWriter{
		file: file,
	}

	return diskFileWriter, nil
}

func (diskFileStore *diskFileStore) RemoveFile(fileName string) error {
	return os.Remove(diskFileStore.fileNameToPath(fileName))
}

func (diskFileStore *diskFileStore) fileNameToPath(fileName string) string {
	return filepath.Join(basePath, fileName)
}

type diskFileReader struct {
	file        *os.File
	contentType string
}

func (diskFileReader *diskFileReader) ContentType() string {
	return diskFileReader.contentType
}

func (diskFileReader *diskFileReader) Size() (int, error) {
	fileInfo, err := diskFileReader.file.Stat()
	if err != nil {
		return -1, err
	}

	fileSizeInt64 := fileInfo.Size()

	if fileSizeInt64 > math.MaxInt32 {
		return -1, errFileTooBig
	}

	return int(fileSizeInt64), nil
}

func (diskFileReader *diskFileReader) ModTime() time.Time {
	fi, err := diskFileReader.file.Stat()
	if err != nil {
		return time.Time{}
	}

	return fi.ModTime()
}

func (diskFileReader *diskFileReader) Read(p []byte) (int, error) {
	return diskFileReader.file.Read(p)
}

func (diskFileReader *diskFileReader) Seek(offset int64, whence int) (int64, error) {
	return diskFileReader.file.Seek(offset, whence)
}

func (diskFileReader *diskFileReader) Close() (err error) {
	err = diskFileReader.file.Close()

	if err == nil {
		diskFileReader.file = nil
	}

	return
}

type diskFileWriter struct {
	file *os.File
}

func (diskFileWriter *diskFileWriter) Write(p []byte) (int, error) {
	bytesWritten, err := diskFileWriter.file.Write(p)
	if err != nil {
		return bytesWritten, err
	}

	// the data written must be available for reading immediately as Write returns
	err = diskFileWriter.file.Sync()

	return bytesWritten, err
}

func (diskFileWriter *diskFileWriter) Close() (err error) {
	err = diskFileWriter.file.Close()

	if err == nil {
		diskFileWriter.file = nil
	}

	return
}

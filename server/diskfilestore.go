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

func (dfs *diskFileStore) GetFileReader(fileName string) (fileReader, error) {
	file, err := os.Open(dfs.fileNameToPath(fileName))
	if err != nil {
		return nil, err
	}

	diskFileReader := &diskFileReader{
		file:        file,
		contentType: contentTypeFromFileName(fileName),
	}

	return diskFileReader, nil
}

func (dfs *diskFileStore) GetFileWriter(fileName string) (fileWriter, error) {
	file, err := os.Create(dfs.fileNameToPath(fileName))
	if err != nil {
		return nil, err
	}

	diskFileWriter := &diskFileWriter{
		file: file,
	}

	return diskFileWriter, nil
}

func (dfs *diskFileStore) RemoveFile(fileName string) error {
	return os.Remove(dfs.fileNameToPath(fileName))
}

func (dfs *diskFileStore) fileNameToPath(fileName string) string {
	return filepath.Join(basePath, fileName)
}

type diskFileReader struct {
	file        *os.File
	contentType string
}

func (dfr *diskFileReader) ContentType() string {
	return dfr.contentType
}

func (dfr *diskFileReader) Size() (int, error) {
	fileInfo, err := dfr.file.Stat()
	if err != nil {
		return -1, err
	}

	fileSizeInt64 := fileInfo.Size()

	if fileSizeInt64 > math.MaxInt32 {
		return -1, errFileTooBig
	}

	return int(fileSizeInt64), nil
}

func (dfr *diskFileReader) ModTime() time.Time {
	fi, err := dfr.file.Stat()
	if err != nil {
		return time.Time{}
	}

	return fi.ModTime()
}

func (dfr *diskFileReader) Read(p []byte) (int, error) {
	return dfr.file.Read(p)
}

func (dfr *diskFileReader) Seek(offset int64, whence int) (int64, error) {
	return dfr.file.Seek(offset, whence)
}

func (dfr *diskFileReader) Close() (err error) {
	err = dfr.file.Close()

	if err == nil {
		dfr.file = nil
	}

	return
}

type diskFileWriter struct {
	file *os.File
}

func (dfw *diskFileWriter) Write(p []byte) (int, error) {
	bytesWritten, err := dfw.file.Write(p)
	if err != nil {
		return bytesWritten, err
	}

	// the data written must be available for reading immediately as Write returns
	err = dfw.file.Sync()

	return bytesWritten, err
}

func (dfw *diskFileWriter) Close() (err error) {
	err = dfw.file.Close()

	if err == nil {
		dfw.file = nil
	}

	return
}

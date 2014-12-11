package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pavben/InstantShare/server/timeout"
)

var (
	ErrAlreadyUploading = errors.New("That file is already uploading or failed")
	ErrNoPreparedUpload = errors.New("No prepared upload with this filename")
	ErrUploadAborted    = errors.New("Upload aborted")
)

type ActiveFileManager struct {
	activeFiles map[string]*ActiveFile
	fileStore   FileStore

	sync.RWMutex
}

type activeFileState int

const (
	activeFileStateNew activeFileState = iota
	activeFileStateAborted
	activeFileStateFinished
)

type ActiveFile struct {
	fileName          string
	currentUpload     *currentUpload
	readLocker        sync.Locker
	dataAvailableCond *sync.Cond
	timeout           timeout.Timeout
	userKey           string
	state             activeFileState

	sync.RWMutex
}

type currentUpload struct {
	bytesWritten   int
	totalFileBytes int
}

func NewActiveFileManager(fileStore FileStore) *ActiveFileManager {
	return &ActiveFileManager{
		activeFiles: make(map[string]*ActiveFile),
		fileStore:   fileStore,
	}
}

func (self *ActiveFileManager) PrepareUpload(fileExtension string, userKey string) string {
	self.Lock()
	defer self.Unlock()

	for {
		fileName := GenerateRandomString() + "." + fileExtension

		_, exists := self.activeFiles[fileName]

		if !exists {
			activeFile := &ActiveFile{
				fileName:          fileName,
				currentUpload:     nil,
				readLocker:        nil,
				dataAvailableCond: nil,
				timeout:           nil,
				userKey:           userKey,
				state:             activeFileStateNew,
			}

			activeFile.readLocker = activeFile.RLocker()

			activeFile.dataAvailableCond = sync.NewCond(activeFile.readLocker)

			activeFile.timeout = timeout.New(10*time.Second, func() {
				self.finishActiveFile(activeFile, fileName)
			})

			self.activeFiles[fileName] = activeFile

			return fileName
		}
	}
}

func (self *ActiveFileManager) finishActiveFile(activeFile *ActiveFile, fileName string) {
	activeFile.Lock()
	{
		if activeFile.currentUpload != nil && activeFile.currentUpload.bytesWritten == activeFile.currentUpload.totalFileBytes {
			activeFile.state = activeFileStateFinished
		} else {
			activeFile.state = activeFileStateAborted
		}

		activeFile.dataAvailableCond.Broadcast()
	}
	activeFile.Unlock()

	self.Lock()
	delete(self.activeFiles, fileName)
	self.Unlock()
}

func (self *ActiveFileManager) Upload(fileName string, fileData io.ReadCloser, contentLength int, userKey string) (err error) {
	// prepare upload
	activeFile, err := func() (*ActiveFile, error) {
		self.Lock()

		defer self.Unlock()

		if activeFile, exists := self.activeFiles[fileName]; exists {
			activeFile.Lock()

			defer activeFile.Unlock()

			if activeFile.currentUpload == nil {
				activeFile.currentUpload = &currentUpload{
					bytesWritten:   -1,
					totalFileBytes: contentLength,
				}

				return activeFile, nil
			} else {

				return nil, ErrAlreadyUploading
			}
		} else {
			return nil, ErrNoPreparedUpload
		}
	}()

	if err != nil {
		return err
	}

	fileWriter, err := self.fileStore.GetFileWriter(fileName)

	if err != nil {
		return err
	}

	defer func() {
		fileWriter.Close()

		if err != nil {
			self.fileStore.RemoveFile(fileName)
		}

		activeFile.timeout.Cancel()
		self.finishActiveFile(activeFile, fileName)
	}()

	// now that the file has been created, indicate that by setting bytesWritten to 0
	func() {
		self.Lock()

		defer self.Unlock()

		activeFile.currentUpload.bytesWritten = 0
	}()

	activeFile.dataAvailableCond.Broadcast()

	//buf := make([]byte, 250000)
	buf := make([]byte, 25000)

	for {
		//time.Sleep(2 * time.Second) // HACK
		time.Sleep(3 * time.Millisecond) // HACK
		bytesRead, err := fileData.Read(buf)

		if bytesRead > 0 {
			activeFile.timeout.Reset()

			_, err := fileWriter.Write(buf[:bytesRead])

			if err != nil {
				return err
			}

			func() {
				activeFile.Lock()
				defer activeFile.Unlock()

				activeFile.currentUpload.bytesWritten += bytesRead
			}()

			activeFile.dataAvailableCond.Broadcast()
		}

		if err != nil {
			if err == io.EOF {
				// done reading/writing
				// save the file to the database and remove it from activeFileManager

				log.Println("Done uploading file")

				// no need to remove it from ActiveFileManager since the timeout will do that

				return nil
			} else {
				// non-EOF error
				return err
			}
		}
	}
}

func (self *ActiveFileManager) GetReaderForFileName(fileName string) FileReader {
	activeFile := func() *ActiveFile {
		self.RLock()

		defer self.RUnlock()

		if activeFile, exists := self.activeFiles[fileName]; exists {
			return activeFile
		} else {
			return nil
		}
	}()

	if activeFile == nil {
		return nil
	}

	return activeFile.GetReader(self.fileStore)
}

// This will block until activeFile.currentUpload is set and the file writer has been created
func (self *ActiveFile) GetReader(fileStore FileStore) FileReader {
	self.readLocker.Lock()

	defer self.readLocker.Unlock()

	if self.state == activeFileStateAborted {
		return nil
	}

	// wait until the file is created
	for self.currentUpload == nil || self.currentUpload.bytesWritten < 0 {
		self.dataAvailableCond.Wait()

		if self.state == activeFileStateAborted {
			return nil
		}
	}

	fileReader, err := fileStore.GetFileReader(self.fileName)

	if err != nil {
		return nil
	}

	return &ActiveFileReader{
		activeFile: self,
		fileReader: fileReader,
	}
}

type ActiveFileReader struct {
	activeFile *ActiveFile
	fileReader FileReader
	//bytesRead  int
	seekPos int64

	sync.Mutex
}

func (self *ActiveFileReader) ContentType() string {
	return ContentTypeFromFileName(self.activeFile.fileName)
}

func (self *ActiveFileReader) Size() (int, error) {
	return self.activeFile.currentUpload.totalFileBytes, nil
}

func (self *ActiveFileReader) ModTime() time.Time {
	return time.Time{}
}

func (self *ActiveFileReader) Seek(offset int64, whence int) (int64, error) {
	self.activeFile.readLocker.Lock()
	defer self.activeFile.readLocker.Unlock()

	fmt.Println("Seek:", offset, whence)

	switch whence {
	case os.SEEK_SET:
		self.seekPos = offset
	case os.SEEK_CUR:
		self.seekPos += offset
	case os.SEEK_END:
		self.seekPos = int64(self.activeFile.currentUpload.totalFileBytes) - offset
	}

	_, err := self.fileReader.Seek(offset, whence)
	if err != nil {
		panic(err)
		return -1, err
	}

	// TODO: Return errors when needed.
	fmt.Println("Seek return:", self.seekPos)
	return self.seekPos, nil
}

func (self *ActiveFileReader) Read(p []byte) (n int, err error) {
	time.Sleep(9 * time.Millisecond)

	self.activeFile.readLocker.Lock()
	defer self.activeFile.readLocker.Unlock()

	fmt.Println("Read:", len(p), "seekPos:", self.seekPos)
	defer func() {
		fmt.Println("Read (n, err):", n, err)
	}()

	if self.activeFile.state == activeFileStateAborted {
		return 0, ErrUploadAborted
	}

	// if done reading
	// TODO: Maybe error if > totalFileBytes.
	if self.seekPos >= int64(self.activeFile.currentUpload.totalFileBytes) {
		return 0, io.EOF
	}

	// wait until there is more data to read
	for self.seekPos >= int64(self.activeFile.currentUpload.bytesWritten) {
		self.activeFile.dataAvailableCond.Wait()

		if self.activeFile.state == activeFileStateAborted {
			return 0, ErrUploadAborted
		}
	}

	n, err = self.fileReader.Read(p)

	self.seekPos += int64(n)

	// clear the error if it's EOF since it won't be EOF when more data is written
	if err == io.EOF {
		err = nil
	}

	return n, err
}

func (self *ActiveFileReader) Close() error {
	return self.fileReader.Close()
}

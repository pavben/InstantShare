package main

import (
	"errors"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

var (
	ErrAlreadyUploading = errors.New("That file is already uploading or failed")
	ErrInvalidFileId    = errors.New("Invalid File ID")
	ErrUploadAborted    = errors.New("Upload aborted")
)

type ActiveFileManager struct {
	activeFiles map[string]*ActiveFile

	sync.RWMutex
}

type ActiveFile struct {
	fileName          string
	currentUpload     *currentUpload
	readLocker        sync.Locker
	dataAvailableCond *sync.Cond
	userKey           string
	aborted           bool

	sync.RWMutex
}

type ActiveFileReader struct {
	activeFile *ActiveFile
	file       *os.File
	bytesRead  int

	sync.Mutex
}

type currentUpload struct {
	contentType    string
	bytesWritten   int
	totalFileBytes int
}

func NewActiveFileManager() *ActiveFileManager {
	return &ActiveFileManager{
		activeFiles: make(map[string]*ActiveFile),
	}
}

func (self *ActiveFileManager) PrepareUpload(fileExtension string, userKey string) string {
	self.Lock()

	defer self.Unlock()

	for {
		fileName := GenerateRandomString() + "." + fileExtension

		fileName = "file.png" // HACK

		_, exists := self.activeFiles[fileName]

		if !exists {
			activeFile := &ActiveFile{
				fileName:          fileName,
				currentUpload:     nil,
				readLocker:        nil,
				dataAvailableCond: nil,
				userKey:           userKey,
			}

			activeFile.readLocker = activeFile.RLocker()

			activeFile.dataAvailableCond = sync.NewCond(activeFile.readLocker)

			self.activeFiles[fileName] = activeFile

			return fileName
		}
	}
}

func (self *ActiveFileManager) Upload(fileName string, fileData io.ReadCloser, contentType string, contentLength int, userKey string) (err error) {
	filePath := FileNameToPath(fileName)

	// prepare upload
	activeFile, err := func() (*ActiveFile, error) {
		self.Lock()

		defer self.Unlock()

		if activeFile, exists := self.activeFiles[fileName]; exists {
			if activeFile.currentUpload == nil {
				activeFile.currentUpload = &currentUpload{
					contentType:    contentType,
					bytesWritten:   0,
					totalFileBytes: contentLength,
				}

				return activeFile, nil
			} else {
				return nil, ErrAlreadyUploading
			}
		} else {
			return nil, ErrInvalidFileId
		}
	}()

	if err != nil {
		return err
	}

	outputFile, err := os.Create(filePath)

	if err != nil {
		return err
	}

	defer func() {
		outputFile.Close()

		if err != nil {
			os.Remove(filePath)
		}

		activeFile.aborted = true
	}()

	buf := make([]byte, 2)

	for {
		log.Println("Waiting..")
		time.Sleep(2 * time.Second) // HACK
		bytesRead, err := fileData.Read(buf)

		log.Println("bytesRead", bytesRead, "err", err)

		if bytesRead > 0 {
			log.Printf("Writing [%v]\n", buf[:bytesRead])
			_, err := outputFile.Write(buf[:bytesRead])

			log.Println("Write result, err =", err)

			if err != nil {
				return err
			}

			// flush the chunk to disk so that it can be accessed by readers immediately after we notify
			err = outputFile.Sync()

			if err != nil {
				return err
			}

			activeFile.currentUpload.bytesWritten += bytesRead

			activeFile.dataAvailableCond.Broadcast()
		}

		if err != nil {
			if err == io.EOF {
				// done reading/writing
				// save the file to the database and remove it from activeFileManager

				log.Println("Done uploading file")

				func() {
					self.Lock()

					defer self.Unlock()

					delete(self.activeFiles, fileName)
				}()

				return nil
			} else {
				// non-EOF error
				return err
			}
		}
	}
}

func (self *ActiveFileManager) GetReaderForFileName(fileName string) io.ReadCloser {
	self.RLock()

	defer self.RUnlock()

	if activeFile, exists := self.activeFiles[fileName]; exists {
		return activeFile.GetReader()
	} else {
		return nil
	}
}

func (self *ActiveFile) GetReader() io.ReadCloser {
	return &ActiveFileReader{
		activeFile: self,
		file:       nil,
		bytesRead:  0,
	}
}

func (self *ActiveFileReader) Read(p []byte) (int, error) {
	self.activeFile.readLocker.Lock()

	defer self.activeFile.readLocker.Unlock()

	checkAborted := func() bool {
		if self.activeFile.aborted {
			self.Close()
			return true
		} else {
			return false
		}
	}

	if checkAborted() {
		return 0, ErrUploadAborted
	}

	// wait until the file starts being written
	for self.activeFile.currentUpload == nil {
		self.activeFile.dataAvailableCond.Wait()

		if checkAborted() {
			return 0, ErrUploadAborted
		}
	}

	if self.file == nil {
		// TODO: auto-create "files"
		file, err := os.Open(FileNameToPath(self.activeFile.fileName))

		if err != nil {
			return 0, err
		}

		self.file = file
	}

	// if done reading
	if self.bytesRead >= self.activeFile.currentUpload.totalFileBytes {
		return 0, io.EOF
	}

	// wait until there is more data to read
	for self.bytesRead >= self.activeFile.currentUpload.bytesWritten {
		self.activeFile.dataAvailableCond.Wait()

		if checkAborted() {
			return 0, ErrUploadAborted
		}
	}

	n, err := self.file.Read(p)

	self.bytesRead += n

	// clear the error if it's EOF since it won't be EOF when more data is written
	if err == io.EOF {
		err = nil
	}

	return n, err
}

func (self *ActiveFileReader) Close() error {
	if self.file != nil {
		return self.file.Close()
	} else {
		return nil
	}
}

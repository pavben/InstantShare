package main

import (
	"errors"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pavben/InstantShare/server/timeout"
)

var (
	errAlreadyUploading = errors.New("that file is already uploading or failed")
	errNoPreparedUpload = errors.New("no prepared upload with this filename")
	errUploadAborted    = errors.New("upload aborted")
)

type activeFileManager struct {
	activeFiles map[string]*activeFile
	fileStore   fileStore

	sync.RWMutex
}

type activeFileState int

const (
	activeFileStateNew activeFileState = iota
	activeFileStateAborted
	activeFileStateFinished
)

type activeFile struct {
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

func newActiveFileManager(fileStore fileStore) *activeFileManager {
	return &activeFileManager{
		activeFiles: make(map[string]*activeFile),
		fileStore:   fileStore,
	}
}

func (activeFileManager *activeFileManager) PrepareUpload(fileExtension string, userKey string) string {
	activeFileManager.Lock()
	defer activeFileManager.Unlock()

	for {
		fileName := generateRandomString()
		if fileExtension != "" {
			fileName += "." + fileExtension
		}

		_, exists := activeFileManager.activeFiles[fileName]
		if !exists {
			activeFile := &activeFile{
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
				activeFileManager.finishActiveFile(activeFile, fileName)
			})
			activeFileManager.activeFiles[fileName] = activeFile

			return fileName
		}
	}
}

func (activeFileManager *activeFileManager) finishActiveFile(activeFile *activeFile, fileName string) {
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

	activeFileManager.Lock()
	delete(activeFileManager.activeFiles, fileName)
	activeFileManager.Unlock()
}

func (activeFileManager *activeFileManager) Upload(fileName string, fileData io.ReadCloser, contentLength int, userKey string) (err error) {
	// prepare upload
	activeFile, err := func() (*activeFile, error) {
		activeFileManager.Lock()
		defer activeFileManager.Unlock()

		activeFile, exists := activeFileManager.activeFiles[fileName]
		if !exists {
			return nil, errNoPreparedUpload
		}

		activeFile.Lock()
		defer activeFile.Unlock()

		if activeFile.currentUpload != nil {
			return nil, errAlreadyUploading
		}

		activeFile.currentUpload = &currentUpload{
			bytesWritten:   -1,
			totalFileBytes: contentLength,
		}
		return activeFile, nil
	}()
	if err != nil {
		return err
	}

	fileWriter, err := activeFileManager.fileStore.GetFileWriter(fileName)
	if err != nil {
		return err
	}

	defer func() {
		fileWriter.Close()

		// If Upload failed and is returning a non-nil error, then remove the file we created here (inside GetFileWriter).
		if err != nil {
			activeFileManager.fileStore.RemoveFile(fileName)
		}

		activeFile.timeout.Cancel()
		activeFileManager.finishActiveFile(activeFile, fileName)
	}()

	// now that the file has been created, indicate that by setting bytesWritten to 0
	func() {
		activeFileManager.Lock()

		defer activeFileManager.Unlock()

		activeFile.currentUpload.bytesWritten = 0
	}()

	activeFile.dataAvailableCond.Broadcast()

	buf := make([]byte, 250000)

	for {
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

				// no need to remove it from activeFileManager since the timeout will do that

				return nil
			}
			// non-EOF error
			return err
		}
	}
}

func (activeFileManager *activeFileManager) GetReaderForFileName(fileName string) fileReader {
	activeFile := func() *activeFile {
		activeFileManager.RLock()

		defer activeFileManager.RUnlock()

		if activeFile, exists := activeFileManager.activeFiles[fileName]; exists {
			return activeFile
		}
		return nil
	}()

	if activeFile == nil {
		return nil
	}

	return activeFile.GetReader(activeFileManager.fileStore)
}

// This will block until activeFile.currentUpload is set and the file writer has been created
func (activeFile *activeFile) GetReader(fileStore fileStore) fileReader {
	activeFile.readLocker.Lock()

	defer activeFile.readLocker.Unlock()

	if activeFile.state == activeFileStateAborted {
		return nil
	}

	// wait until the file is created
	for activeFile.currentUpload == nil || activeFile.currentUpload.bytesWritten < 0 {
		activeFile.dataAvailableCond.Wait()

		if activeFile.state == activeFileStateAborted {
			return nil
		}
	}

	fileReader, err := fileStore.GetFileReader(activeFile.fileName)
	if err != nil {
		return nil
	}

	return &activeFileReader{
		activeFile: activeFile,
		fileReader: fileReader,
	}
}

type activeFileReader struct {
	activeFile *activeFile
	fileReader fileReader
	seekPos    int64

	sync.Mutex
}

func (activeFileReader *activeFileReader) ContentType() string {
	return contentTypeFromFileName(activeFileReader.activeFile.fileName)
}

func (activeFileReader *activeFileReader) Size() (int, error) {
	return activeFileReader.activeFile.currentUpload.totalFileBytes, nil
}

func (activeFileReader *activeFileReader) ModTime() time.Time {
	return time.Time{}
}

func (activeFileReader *activeFileReader) Seek(offset int64, whence int) (int64, error) {
	activeFileReader.activeFile.readLocker.Lock()
	defer activeFileReader.activeFile.readLocker.Unlock()

	switch whence {
	case os.SEEK_SET:
		activeFileReader.seekPos = offset
	case os.SEEK_CUR:
		activeFileReader.seekPos += offset
	case os.SEEK_END:
		activeFileReader.seekPos = int64(activeFileReader.activeFile.currentUpload.totalFileBytes) - offset
	}

	_, err := activeFileReader.fileReader.Seek(offset, whence)
	if err != nil {
		return -1, err
	}

	// TODO: Return errors when needed.
	return activeFileReader.seekPos, nil
}

func (activeFileReader *activeFileReader) Read(p []byte) (n int, err error) {
	time.Sleep(9 * time.Millisecond)

	activeFileReader.activeFile.readLocker.Lock()
	defer activeFileReader.activeFile.readLocker.Unlock()

	if activeFileReader.activeFile.state == activeFileStateAborted {
		return 0, errUploadAborted
	}

	// if done reading
	// TODO: Maybe error if > totalFileBytes.
	if activeFileReader.seekPos >= int64(activeFileReader.activeFile.currentUpload.totalFileBytes) {
		return 0, io.EOF
	}

	// wait until there is more data to read
	for activeFileReader.seekPos >= int64(activeFileReader.activeFile.currentUpload.bytesWritten) {
		activeFileReader.activeFile.dataAvailableCond.Wait()

		if activeFileReader.activeFile.state == activeFileStateAborted {
			return 0, errUploadAborted
		}
	}

	n, err = activeFileReader.fileReader.Read(p)

	activeFileReader.seekPos += int64(n)

	// clear the error if it's EOF since it won't be EOF when more data is written
	if err == io.EOF {
		err = nil
	}

	return n, err
}

func (activeFileReader *activeFileReader) Close() error {
	return activeFileReader.fileReader.Close()
}

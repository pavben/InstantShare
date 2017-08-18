package main

import (
	"errors"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pavben/InstantShare/id"
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

func (afm *activeFileManager) PrepareUpload(fileExtension string, userKey string) (string, error) {
	afm.Lock()
	defer afm.Unlock()

	for {
		fileName, err := id.Generate()
		if err != nil {
			return "", err
		}
		if fileExtension != "" {
			fileName += "." + fileExtension
		}

		_, exists := afm.activeFiles[fileName]
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
				afm.finishActiveFile(activeFile, fileName)
			})
			afm.activeFiles[fileName] = activeFile

			return fileName, nil
		}
	}
}

func (afm *activeFileManager) finishActiveFile(activeFile *activeFile, fileName string) {
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

	afm.Lock()
	delete(afm.activeFiles, fileName)
	afm.Unlock()
}

func (afm *activeFileManager) Upload(fileName string, fileData io.ReadCloser, contentLength int, userKey string) (err error) {
	// prepare upload
	activeFile, err := func() (*activeFile, error) {
		afm.Lock()
		defer afm.Unlock()

		activeFile, exists := afm.activeFiles[fileName]
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

	fileWriter, err := afm.fileStore.GetFileWriter(fileName)
	if err != nil {
		return err
	}

	defer func() {
		fileWriter.Close()

		// If Upload failed and is returning a non-nil error, then remove the file we created here (inside GetFileWriter).
		if err != nil {
			afm.fileStore.RemoveFile(fileName)
		}

		activeFile.timeout.Cancel()
		afm.finishActiveFile(activeFile, fileName)
	}()

	// now that the file has been created, indicate that by setting bytesWritten to 0
	func() {
		afm.Lock()

		defer afm.Unlock()

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

func (afm *activeFileManager) GetReaderForFileName(fileName string) fileReader {
	activeFile := func() *activeFile {
		afm.RLock()

		defer afm.RUnlock()

		if activeFile, exists := afm.activeFiles[fileName]; exists {
			return activeFile
		}
		return nil
	}()

	if activeFile == nil {
		return nil
	}

	return activeFile.GetReader(afm.fileStore)
}

// This will block until af.currentUpload is set and the file writer has been created
func (af *activeFile) GetReader(fileStore fileStore) fileReader {
	af.readLocker.Lock()

	defer af.readLocker.Unlock()

	if af.state == activeFileStateAborted {
		return nil
	}

	// wait until the file is created
	for af.currentUpload == nil || af.currentUpload.bytesWritten < 0 {
		af.dataAvailableCond.Wait()

		if af.state == activeFileStateAborted {
			return nil
		}
	}

	fileReader, err := fileStore.GetFileReader(af.fileName)
	if err != nil {
		return nil
	}

	return &activeFileReader{
		activeFile: af,
		fileReader: fileReader,
	}
}

type activeFileReader struct {
	activeFile *activeFile
	fileReader fileReader
	seekPos    int64

	sync.Mutex
}

func (afr *activeFileReader) ContentType() string {
	return contentTypeFromFileName(afr.activeFile.fileName)
}

func (afr *activeFileReader) Size() (int, error) {
	return afr.activeFile.currentUpload.totalFileBytes, nil
}

func (afr *activeFileReader) ModTime() time.Time {
	return time.Time{}
}

func (afr *activeFileReader) Seek(offset int64, whence int) (int64, error) {
	afr.activeFile.readLocker.Lock()
	defer afr.activeFile.readLocker.Unlock()

	switch whence {
	case os.SEEK_SET:
		afr.seekPos = offset
	case os.SEEK_CUR:
		afr.seekPos += offset
	case os.SEEK_END:
		afr.seekPos = int64(afr.activeFile.currentUpload.totalFileBytes) - offset
	}

	_, err := afr.fileReader.Seek(offset, whence)
	if err != nil {
		return -1, err
	}

	// TODO: Return errors when needed.
	return afr.seekPos, nil
}

func (afr *activeFileReader) Read(p []byte) (n int, err error) {
	time.Sleep(9 * time.Millisecond)

	afr.activeFile.readLocker.Lock()
	defer afr.activeFile.readLocker.Unlock()

	if afr.activeFile.state == activeFileStateAborted {
		return 0, errUploadAborted
	}

	// if done reading
	// TODO: Maybe error if > totalFileBytes.
	if afr.seekPos >= int64(afr.activeFile.currentUpload.totalFileBytes) {
		return 0, io.EOF
	}

	// wait until there is more data to read
	for afr.seekPos >= int64(afr.activeFile.currentUpload.bytesWritten) {
		afr.activeFile.dataAvailableCond.Wait()

		if afr.activeFile.state == activeFileStateAborted {
			return 0, errUploadAborted
		}
	}

	n, err = afr.fileReader.Read(p)

	afr.seekPos += int64(n)

	// clear the error if it's EOF since it won't be EOF when more data is written
	if err == io.EOF {
		err = nil
	}

	return n, err
}

func (afr *activeFileReader) Close() error {
	return afr.fileReader.Close()
}

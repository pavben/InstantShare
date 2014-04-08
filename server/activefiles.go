package main

import (
	"io"
	"os"
	"sync"

	"github.com/shurcooL/go-goon"
)

type ActiveFileManager struct {
	activeFiles map[FileId]*ActiveFile
	mutex       sync.RWMutex
}

type ActiveFile struct {
	currentUpload *currentUpload
	userKey       string
}

type currentUpload struct {
	fileSource     *io.Writer
	file           *os.File
	contentType    string
	fileExtension  string
	bytesWritten   int
	totalFileBytes int
}

func NewActiveFileManager() *ActiveFileManager {
	return &ActiveFileManager{
		activeFiles: make(map[FileId]*ActiveFile),
	}
}

func (self *ActiveFileManager) PrepareUpload(generateFileIdFunc func() FileId, userKey string) FileId {
	self.mutex.Lock()

	defer self.mutex.Unlock()

	for {
		fileId := generateFileIdFunc()

		_, exists := self.activeFiles[fileId]

		if !exists {
			self.activeFiles[fileId] = &ActiveFile{
				currentUpload: nil,
				userKey:       userKey,
			}

			goon.Dump(self.activeFiles)

			return fileId
		}
	}
}

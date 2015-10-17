// Command server is the Instant Share server. It is responsible for accepting uploads and serving those files.
package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
)

const maxFileSize = 200 * 1024 * 1024

func main() {
	fileStore, err := newDiskFileStore()
	if err != nil {
		log.Println(err)
		return
	}

	activeFileManager := newActiveFileManager(fileStore)

	webHandler := getWebHandler(activeFileManager, fileStore)

	err = http.ListenAndServe(":27080", webHandler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func getWebHandler(activeFileManager *activeFileManager, fileStore fileStore) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		method := req.Method
		path := urlPathToArray(req.URL.Path)

		switch {
		case len(path) == 1:
			if method == "GET" {
				// request for a file
				fileName := path[0]
				fileReader := getReaderForFileName(fileName, activeFileManager, fileStore)
				if fileReader == nil {
					http.NotFound(res, req)
					return
				}
				defer fileReader.Close()
				// stream the fileReader to the response
				res.Header().Set("Content-Type", fileReader.ContentType())
				http.ServeContent(res, req, "", fileReader.ModTime(), fileReader)
			} else if method == "PUT" {
				// uploading a file
				handlePutFile(res, req, path[0], activeFileManager)
			} else {
				http.Error(res, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
		case len(path) == 2 && path[0] == "api" && path[1] == "getfilename" && method == "GET":
			fileExtension := req.URL.Query().Get("ext")

			newFilename := activeFileManager.PrepareUpload(fileExtension, "USERKEYTODO")

			log.Println("/api/getfilename returning", newFilename)

			res.Write([]byte(newFilename))
		default:
			http.NotFound(res, req)
		}
	})
}

func handlePutFile(res http.ResponseWriter, req *http.Request, fileName string, activeFileManager *activeFileManager) {
	contentType := req.Header.Get("Content-Type")

	if contentType == "" {
		http.Error(res, "Bad Request: Missing required Content-Type header", http.StatusBadRequest)
		return
	}

	if req.ContentLength < 1 {
		http.Error(res, "Bad Request: Content-Length is required and must be positive", http.StatusBadRequest)
		return
	}

	if req.ContentLength >= maxFileSize {
		http.Error(res, "Bad Request: File to upload exceeds "+strconv.Itoa(maxFileSize), http.StatusBadRequest)
		return
	}

	err := activeFileManager.Upload(fileName, req.Body, int(req.ContentLength), "USERKEYTODO")
	if err != nil {
		http.Error(res, "Error: "+err.Error(), http.StatusInternalServerError)
	}
}

func getReaderForFileName(fileName string, activeFileManager *activeFileManager, fileStore fileStore) fileReader {
	fileReader := activeFileManager.GetReaderForFileName(fileName)

	if fileReader != nil {
		return fileReader
	}

	fileReader, err := fileStore.GetFileReader(fileName)
	if err != nil {
		return nil
	}

	return fileReader
}

func urlPathToArray(path string) []string {
	splitPath := strings.Split(path, "/")

	startIdx := 0

	if len(splitPath) >= 1 && splitPath[startIdx] == "" {
		startIdx++
	}

	endIdx := len(splitPath) - 1

	if len(splitPath) >= 1 && splitPath[endIdx] == "" {
		endIdx--
	}

	return splitPath[startIdx : endIdx+1]
}

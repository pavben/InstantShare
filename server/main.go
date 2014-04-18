package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const maxFileSize = 200 * 1024 * 1024

var (
	fileStorePathFlag = flag.String("file-store-path", "", "Path to dir for uploaded files; you must provide an existing dir.")
)

func main() {
	flag.Parse()

	// Check for mandatory flag value.
	if *fileStorePathFlag == "" {
		flag.Usage()
		os.Exit(2)
	}

	fileStore, err := NewDiskFileStore(*fileStorePathFlag)

	if err != nil {
		log.Println(err)
		return
	}

	activeFileManager := NewActiveFileManager(fileStore)

	webHandler := getWebHandler(activeFileManager, fileStore)

	err = http.ListenAndServe(":27080", webHandler)

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

	// TODO: TLS
	err = http.ListenAndServe(":27443", webHandler)

	if err != nil {
		log.Fatal("ListenAndServeTLS: ", err)
	}
}

func getWebHandler(activeFileManager *ActiveFileManager, fileStore FileStore) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		method := req.Method
		path := urlPathToArray(req.URL.Path)

		switch {
		case len(path) == 1:
			if method == "GET" {
				//goon.Dump(req)
				// request for a file

				fileName := path[0]

				fileReader := getReaderForFileName(fileName, activeFileManager, fileStore)

				if fileReader == nil {
					http.NotFound(res, req)
					return
				}

				defer fileReader.Close()

				// stream the fileReader to the response

				res.Header().Add("Content-Type", fileReader.ContentType())

				fileSize, err := fileReader.Size()

				if err != nil {
					http.NotFound(res, req) // only happens if the upload was aborted
					return
				}

				res.Header().Add("Content-Length", strconv.Itoa(fileSize))

				buf := make([]byte, 1024) // TODO: real buffer size

				for {
					bytesRead, err := fileReader.Read(buf)

					if bytesRead > 0 {
						res.Write(buf[:bytesRead])
					}

					if err != nil {
						// whether the error is EOF or something else, stop streaming

						// TODO: how I end the request?
						return
					}
				}
			} else if method == "PUT" {
				// uploading a file
				handlePutFile(res, req, path[0], activeFileManager)
			} else {
				http.Error(res, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
		case len(path) == 2 && path[0] == "api" && path[1] == "getfilename" && method == "GET":
			//goon.Dump(req)

			query, err := url.ParseQuery(req.URL.RawQuery)

			if err != nil {
				http.Error(res, "Bad Request: Invalid query parameters", http.StatusBadRequest)
				return
			}

			fileExtension, fileExtensionProvided := query["ext"]

			if !fileExtensionProvided {
				http.Error(res, "Bad Request: Missing file extension parameter", http.StatusBadRequest)
				return
			}

			// CHECK: Can len(fileExtension) == 0?
			if len(fileExtension) < 1 {
				return
			}

			newFilename := activeFileManager.PrepareUpload(fileExtension[0], "USERKEYTODO")

			log.Println("/api/getfilename returning", newFilename)

			res.Write([]byte(newFilename))
		default:
			http.NotFound(res, req)
		}
	})
}

func handlePutFile(res http.ResponseWriter, req *http.Request, fileName string, activeFileManager *ActiveFileManager) {
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

func getReaderForFileName(fileName string, activeFileManager *ActiveFileManager, fileStore FileStore) FileReader {
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
		startIdx += 1
	}

	endIdx := len(splitPath) - 1

	if len(splitPath) >= 1 && splitPath[endIdx] == "" {
		endIdx -= 1
	}

	return splitPath[startIdx : endIdx+1]
}

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/shurcooL/go-goon"
)

const maxFileSize = 200 * 1024 * 1024

func main() {
	activeFileManager := NewActiveFileManager()

	fileStore, err := NewDiskFileStore()

	if err != nil {
		log.Println(err)
		return
	}

	webHandler := getWebHandler(activeFileManager, fileStore)

	/*
		activeFileManager.PrepareUpload(GenerateNewFileID, "USERKEYTODO")

		go func() {
			time.Sleep(10 * time.Second)

			reader := activeFileManager.GetReaderForFileId("fileid")

			log.Println("Got reader", reader)

			for {
				buf := make([]byte, 20, 20)

				log.Println("Reading...")

				n, err := reader.Read(buf)

				log.Printf("READ %d bytes, err = %v, buf = %v\n", n, err, buf)

				if err == io.EOF {
					log.Println("Done reading")
					break
				}
			}
		}()
	*/

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

				reader := getReaderForFileName(fileName, activeFileManager, fileStore)

				if reader == nil {
					http.NotFound(res, req)
					return
				}

				defer reader.Close()

				// stream the reader to the response

				// TODO: content type
				res.Header().Add("Content-Type", "image/jpeg")

				// TODO: length of file

				buf := make([]byte, 1024) // TODO: real buffer size

				for {
					bytesRead, err := reader.Read(buf)

					if bytesRead > 0 {
						fmt.Println("Writing", bytesRead, "bytes to response")
						res.Write(buf[:bytesRead])
						//fl, _ := res.(http.Flusher)

						//fl.Flush()
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
			goon.Dump(req)

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

	err := activeFileManager.Upload(fileName, req.Body, contentType, int(req.ContentLength), "USERKEYTODO")

	if err != nil {
		http.Error(res, "Error: "+err.Error(), http.StatusInternalServerError)
	}
}

func getReaderForFileName(fileName string, activeFileManager *ActiveFileManager, fileStore FileStore) io.ReadCloser {
	activeFileReader := activeFileManager.GetReaderForFileName(fileName)

	if activeFileReader != nil {
		return activeFileReader
	}

	file := fileStore.GetFile(fileName)

	if file != nil {
		return file
	}

	return nil
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

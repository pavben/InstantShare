package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
)

const maxFileSize = 200 * 1024 * 1024

func main() {
	activeFileManager := NewActiveFileManager()

	webHandler := getWebHandler(activeFileManager)

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

	err := http.ListenAndServe(":27080", webHandler)

	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

	// TODO: TLS
	err = http.ListenAndServe(":27443", webHandler)

	if err != nil {
		log.Fatal("ListenAndServeTLS: ", err)
	}
}

func getWebHandler(activeFileManager *ActiveFileManager) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		method := req.Method
		path := urlPathToArray(req.URL.Path)

		switch {
		case len(path) == 1:
			if method == "GET" {
				// request for a file
			} else if method == "PUT" {
				// uploading a file
				handlePutFile(res, req, path[0], activeFileManager)
			} else {
				http.Error(res, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
		case len(path) == 2 && path[0] == "api" && path[1] == "getid" && method == "GET":
			//goon.Dump(req)

			newFileId := activeFileManager.PrepareUpload(GenerateNewFileID, "USERKEYTODO")

			log.Println("/api/getid returning", newFileId)

			res.Write([]byte(newFileId))
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

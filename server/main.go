package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/shurcooL/go-goon"
)

func main() {
	activeFileManager := NewActiveFileManager()

	webHandler := getWebHandler(activeFileManager)

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
			} else {
				http.Error(res, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
		case len(path) == 2 && path[0] == "api" && path[1] == "getid" && method == "GET":
			goon.Dump(req)

			newFileId := activeFileManager.PrepareUpload(GenerateNewFileID, "USERKEYTODO")

			log.Println("/api/getid returning", newFileId)

			res.Write([]byte(newFileId))
		default:
			http.NotFound(res, req)
		}
	})
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

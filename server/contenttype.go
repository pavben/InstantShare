package main

import (
	"mime"
	"path/filepath"
)

func contentTypeFromFileName(fileName string) string {
	ext := filepath.Ext(fileName)
	if ext == ".mov" {
		// "video/quicktime" is correct, but temporarily using "video/mp4" because it plays in Chrome.
		return "video/mp4"
	}
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}

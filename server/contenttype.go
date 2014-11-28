package main

import "path/filepath"

func ContentTypeFromFileName(fileName string) string {
	switch filepath.Ext(fileName) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".mov":
		// TODO: Handle this correctly. "video/quicktime" is correct, but I'm temporarily using "video/mp4" for testing in Chrome.
		//return "video/quicktime"
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

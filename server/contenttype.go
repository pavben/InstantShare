package main

import "strings"

func ContentTypeFromFileName(fileName string) string {
	var extension string

	dotIndex := strings.LastIndex(fileName, ".")

	if dotIndex != -1 {
		extension = strings.ToLower(fileName[dotIndex+1:])
	}

	switch extension {
	case "png":
		return "image/png"
	case "jpg", "jpeg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}

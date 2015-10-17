package main

import (
	"log"
	"time"

	"github.com/shurcooL/go/u/u4"
	"github.com/shurcooL/trayhost"
)

func debugMenuItems() []trayhost.MenuItem {
	return []trayhost.MenuItem{
		trayhost.MenuItem{
			Title: "Debug: Get Clipboard Content",
			Handler: func() {
				cc, err := trayhost.GetClipboardContent()
				log.Printf("GetClipboardContent() error: %v\n", err)
				log.Printf("Text: %q\n", cc.Text)
				log.Printf("Image: %v len(%v)\n", cc.Image.Kind, len(cc.Image.Bytes))
				log.Printf("Files: len(%v) %v\n", len(cc.Files), cc.Files)
			},
		},
		trayhost.MenuItem{
			Title: "Debug: Set Clipboard Text",
			Handler: func() {
				trayhost.SetClipboardText("http://www.example.com/image.png")
			},
		},
		trayhost.MenuItem{
			Title: "Debug: Notification",
			Handler: func() {
				handler := func() {
					u4.Open("http://www.example.com/image.png")
				}
				notification := trayhost.Notification{Title: "Success", Body: "http://www.example.com/image.png", Timeout: 3 * time.Second, Handler: handler}
				//trayhost.Notification{Title: "Upload Failed", Body: "error description goes here"}.Display()
				if cc, err := trayhost.GetClipboardContent(); err == nil && cc.Image.Kind != "" {
					notification.Image = cc.Image
				}
				notification.Display()
			},
		},
	}
}

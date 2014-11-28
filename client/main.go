package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"image"
	"image/png"

	_ "golang.org/x/image/tiff"

	"github.com/shurcooL/go/u/u4"
	"github.com/shurcooL/trayhost"
)

// TODO: Load from config. Have ability to set config.
var hostFlag = flag.String("host", "", "Target server host.")
var debugFlag = flag.Bool("debug", false, "Adds menu items for debugging purposes.")

var httpClient = &http.Client{Timeout: 3 * time.Second}

var clipboardImage trayhost.Image

func instantShareEnabled() bool {
	//fmt.Println("check if clipboard contains something usable")

	fmt.Println("grab content, content-type of clipboard")

	// TODO: Clean this up, merge with section below maybe?
	if files, err := trayhost.GetClipboardFiles(); err == nil && len(files) == 1 && filepath.Ext(files[0]) == ".png" {
		b, err := ioutil.ReadFile(files[0])
		if err == nil {
			clipboardImage.Kind = trayhost.ImageKindPng
			clipboardImage.ContentType = "image/png"
			clipboardImage.Bytes = b
			return true
		}
	} else if err == nil && len(files) == 1 && filepath.Ext(files[0]) == ".mov" {
		b, err := ioutil.ReadFile(files[0])
		if err == nil {
			clipboardImage.Kind = trayhost.ImageKindMov
			clipboardImage.ContentType = "video/quicktime"
			clipboardImage.Bytes = b
			return true
		}
	}

	var err error
	clipboardImage, err = trayhost.GetClipboardImage()
	if err != nil {
		return false
	}

	return true
}

func instantShareHandler() {
	// Convert image to desired destination format (currently, always PNG).
	// TODO: Maybe not do this for files? What if it's a jpeg.
	var extension = "png"
	var imageData []byte
	switch clipboardImage.Kind {
	case trayhost.ImageKindPng:
		imageData = clipboardImage.Bytes
	case trayhost.ImageKindTiff:
		m, _, err := image.Decode(bytes.NewReader(clipboardImage.Bytes))
		if err != nil {
			log.Panicln("image.Decode:", err)
		}

		var buf bytes.Buffer
		err = png.Encode(&buf, m)
		if err != nil {
			log.Panicln("png.Encode:", err)
		}
		imageData = buf.Bytes()
	case trayhost.ImageKindMov:
		imageData = clipboardImage.Bytes
		extension = "mov"
	default:
		log.Println("unsupported source image kind:", clipboardImage.Kind)
		return
	}

	fmt.Println("request URL")

	// TODO: Perhaps move to specifying content-type instead of file extension. Or, if extension, don't strip the dot?
	resp, err := httpClient.Get(*hostFlag + "/api/getfilename?ext=" + extension)
	if err != nil {
		trayhost.Notification{Title: "Upload Failed", Body: err.Error()}.Display()
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	filename, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		trayhost.Notification{Title: "Upload Failed", Body: err.Error()}.Display()
		log.Println(err)
		return
	}

	fmt.Println("display/put URL in clipboard")

	url := *hostFlag + "/" + string(filename)
	trayhost.SetClipboardString(url)
	trayhost.Notification{
		Title:   "Success",
		Body:    url,
		Image:   clipboardImage,
		Timeout: 3 * time.Second,
		Handler: func() {
			// On click, open the displayed URL.
			u4.Open(url)
		},
	}.Display()

	fmt.Println("upload image in background of size", len(imageData))

	go func() {
		req, err := http.NewRequest("PUT", url, bytes.NewReader(imageData))
		if err != nil {
			log.Println(err)
			return
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
			return
		}
		_ = resp.Body.Close()
		fmt.Println("done")
	}()
}

func main() {
	flag.Parse()
	runtime.LockOSThread()

	menuItems := []trayhost.MenuItem{
		trayhost.MenuItem{
			Title:   "Instant Share",
			Enabled: instantShareEnabled,
			Handler: instantShareHandler,
		},
		trayhost.SeparatorMenuItem(),
		trayhost.MenuItem{
			Title:   "Quit",
			Handler: trayhost.Exit,
		},
	}
	if *debugFlag {
		menuItems = append(menuItems,
			trayhost.SeparatorMenuItem(),
			trayhost.MenuItem{
				Title: "Debug: Get Clipboard String",
				Handler: func() {
					str, err := trayhost.GetClipboardString()
					fmt.Printf("GetClipboardString(): %q %v\n", str, err)
				},
			},
			trayhost.MenuItem{
				Title: "Debug: Get Clipboard Image",
				Handler: func() {
					img, err := trayhost.GetClipboardImage()
					fmt.Printf("GetClipboardImage(): %v len(%v) %v\n", img.Kind, len(img.Bytes), err)
				},
			},
			trayhost.MenuItem{
				Title: "Debug: Get Clipboard Files",
				Handler: func() {
					filenames, err := trayhost.GetClipboardFiles()
					fmt.Printf("GetClipboardFiles(): %v len(%v) %v\n", filenames, len(filenames), err)
				},
			},
			trayhost.MenuItem{
				Title: "Debug: Set Clipboard",
				Handler: func() {
					trayhost.SetClipboardString("http://www.example.com/image.png")
				},
			},
			trayhost.MenuItem{
				Title: "Debug: Notification",
				Handler: func() {
					handler := func() {
						u4.Open("http://www.example.com/image.png")
					}
					notification := trayhost.Notification{Title: "Upload Complete", Body: "http://www.example.com/image.png", Timeout: 3 * time.Second, Handler: handler}
					//trayhost.Notification{Title: "Upload Failed", Body: "error description goes here"}.Display()
					if img, err := trayhost.GetClipboardImage(); err == nil {
						notification.Image = img
					}
					notification.Display()
				},
			},
		)
	}

	// TODO: Create a real icon and bake it into the binary.
	// TODO: Optionally, if non-Retina pixel perfection is required, generate or load 1x image and supply that as a second representation.
	iconData, err := ioutil.ReadFile("./icon@2x.png")
	if err != nil {
		panic(err)
	}

	fmt.Println("Starting.")

	trayhost.Initialize("Instant Share", iconData, menuItems)

	trayhost.EnterLoop()

	fmt.Println("Exiting.")
}

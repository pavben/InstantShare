package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"time"

	"image"
	"image/png"

	_ "code.google.com/p/go.image/tiff"

	"github.com/shurcooL/trayhost"
)

// TODO: Load from config. Have ability to set config.
var hostFlag = flag.String("host", "", "Target server host.")
var debugFlag = flag.Bool("debug", false, "Adds menu items for debugging purposes.")

func instantShareEnabled() bool {
	fmt.Println("check if clipboard contains something usable")

	_, err := trayhost.GetClipboardImage()
	if err != nil {
		return false
	}

	return true
}

func instantShareHandler() {
	fmt.Println("grab content, content-type of clipboard")

	img, err := trayhost.GetClipboardImage()
	if err != nil {
		log.Println(err)
		return
	}

	// Convert image to desired destination format (currently, always PNG).
	var imageData []byte
	switch img.Kind {
	case trayhost.ImageKindPng:
		imageData = img.Bytes
	case trayhost.ImageKindTiff:
		m, _, err := image.Decode(bytes.NewReader(img.Bytes))
		if err != nil {
			log.Panicln("image.Decode:", err)
		}

		var buf bytes.Buffer
		err = png.Encode(&buf, m)
		if err != nil {
			log.Panicln("png.Encode:", err)
		}
		imageData = buf.Bytes()
	default:
		log.Println("unsupported source image kind:", img.Kind)
		return
	}

	fmt.Println("request URL")

	resp, err := (&http.Client{Timeout: 1 * time.Second}).Get(*hostFlag + "/api/getfilename?ext=png")
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
	trayhost.Notification{Title: "Upload Complete", Body: url, Timeout: 3 * time.Second}.Display()

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
				Title: "Debug: Set Clipboard",
				Handler: func() {
					trayhost.SetClipboardString("http://www.example.com/image.png")
				},
			},
			trayhost.MenuItem{
				Title: "Debug: Notification",
				Handler: func() {
					trayhost.Notification{Title: "Upload Complete", Body: "http://www.example.com/image.png", Timeout: 3 * time.Second}.Display()
					//trayhost.Notification{Title: "Upload Failed", Body: "error description goes here"}.Display()
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

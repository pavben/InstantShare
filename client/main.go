package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/shurcooL/go/u/u4"
	"github.com/shurcooL/trayhost"
	_ "golang.org/x/image/tiff"
)

// TODO: Load from config. Have ability to set config.
var hostFlag = flag.String("host", "", "Target server host.")
var debugFlag = flag.Bool("debug", false, "Adds menu items for debugging purposes.")

var httpClient = &http.Client{Timeout: 3 * time.Second}

// TODO: Rename Image to "Media" or so, since it's used for both images and videos of various types?
var clipboardImage trayhost.Image

func instantShareEnabled() bool {
	fmt.Println("grab content, content-type of clipboard")

	cc, err := trayhost.GetClipboardContent()
	if err != nil {
		return false
	}

	switch {
	case len(cc.Files) == 1 && filepath.Ext(cc.Files[0]) == ".png": // Single .png file.
		b, err := ioutil.ReadFile(cc.Files[0])
		if err != nil {
			return false
		}
		clipboardImage.Kind = "png"
		clipboardImage.Bytes = b
		return true
	case len(cc.Files) == 1 && filepath.Ext(cc.Files[0]) == ".mov": // Single .mov file.
		b, err := ioutil.ReadFile(cc.Files[0])
		if err != nil {
			return false
		}
		clipboardImage.Kind = "mov"
		clipboardImage.Bytes = b
		return true
	case cc.Image.Kind != "":
		clipboardImage = cc.Image
		return true
	default:
		return false
	}
}

func instantShareHandler() {
	// Convert image to desired destination format (currently, always PNG).
	// TODO: Maybe not do this for files? What if it's a jpeg.
	var extension string
	var imageData []byte
	switch clipboardImage.Kind {
	case "png":
		imageData = clipboardImage.Bytes
		extension = "png"
	case "tiff":
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
		extension = "png"
	case "mov":
		imageData = clipboardImage.Bytes
		extension = "mov"
	default:
		log.Println("unsupported source image kind:", clipboardImage.Kind)
		return
	}

	fmt.Println("request URL")

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
	trayhost.SetClipboardText(url)
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
				Title: "Debug: Get Clipboard Content",
				Handler: func() {
					cc, err := trayhost.GetClipboardContent()
					fmt.Printf("GetClipboardContent() error: %v\n", err)
					fmt.Printf("Text: %q\n", cc.Text)
					fmt.Printf("Image: %v len(%v)\n", cc.Image.Kind, len(cc.Image.Bytes))
					fmt.Printf("Files: len(%v) %v\n", len(cc.Files), cc.Files)
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
					notification := trayhost.Notification{Title: "Upload Complete", Body: "http://www.example.com/image.png", Timeout: 3 * time.Second, Handler: handler}
					//trayhost.Notification{Title: "Upload Failed", Body: "error description goes here"}.Display()
					if cc, err := trayhost.GetClipboardContent(); err == nil && cc.Image.Kind != "" {
						notification.Image = cc.Image
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

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"

	"github.com/shurcooL/trayhost"
)

func instantShareHandler() {
	fmt.Println("grab content, content-type of clipboard")

	img, err := trayhost.GetClipboardImage()
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("request URL")

	resp, err := http.Get("http://localhost:27080/api/getfilename?ext=png")
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	filename, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("display/put URL in clipboard")

	url := "http://localhost:27080/" + string(filename)
	trayhost.SetClipboardString(url)
	// TODO: Notification? Or not?

	fmt.Println("upload image in background of size", len(img))

	go func() {
		req, err := http.NewRequest("PUT", url, bytes.NewReader(img))
		if err != nil {
			log.Println(err)
			return
		}
		req.Header.Set("Content-Type", "image/png")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println(err)
			return
		}
		_ = resp.Body.Close()
	}()
}

func main() {
	runtime.LockOSThread()

	menuItems := []trayhost.MenuItem{
		trayhost.MenuItem{
			Title:   "Instant Share",
			Handler: instantShareHandler,
		},
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
				fmt.Printf("GetClipboardImage(): len(%v) %v\n", len(img), err)
			},
		},
		trayhost.MenuItem{
			Title: "Debug: Set Clipboard",
			Handler: func() {
				trayhost.SetClipboardString("http://www.example.org/image.png")
			},
		},
		trayhost.SeparatorMenuItem(),
		trayhost.MenuItem{
			Title:   "Quit",
			Handler: trayhost.Exit,
		},
	}

	// TODO: Create a real icon and bake it into the binary.
	iconData, err := ioutil.ReadFile("./icon.png")
	if err != nil {
		panic(err)
	}

	fmt.Println("Starting.")

	trayhost.Initialize("InstantShare", iconData, menuItems)

	trayhost.EnterLoop()

	fmt.Println("Exiting.")
}

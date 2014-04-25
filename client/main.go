package main

import (
	"fmt"
	"io/ioutil"
	"runtime"

	"github.com/shurcooL/trayhost"
)

func main() {
	runtime.LockOSThread()

	menuItems := []trayhost.MenuItem{
		trayhost.MenuItem{
			Title: "Instant Share",
			Handler: func() {
				fmt.Println("TODO: grab content, content-type of clipboard")
				fmt.Println("TODO: request URL")
				fmt.Println("TODO: display/put URL in clipboard")
				fmt.Println("TODO: upload image in background")
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

	trayhost.Initialize("InstantShare", iconData, menuItems)

	trayhost.EnterLoop()
}

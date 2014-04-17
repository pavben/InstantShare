package main

import (
	"fmt"
	"io/ioutil"
	"runtime"

	"github.com/overlordtm/trayhost"
)

// TODO: Factor into trayhost.
func trayhost_NewSeparatorMenuItem() trayhost.MenuItem { return trayhost.MenuItem{Title: ""} }

func main() {
	runtime.LockOSThread()

	menuItems := trayhost.MenuItems{
		trayhost.MenuItem{
			Title: "Instant Share",
			Handler: func() {
				fmt.Println("TODO: grab content, content-type of clipboard")
				fmt.Println("TODO: request URL")
				fmt.Println("TODO: display/put URL in clipboard")
				fmt.Println("TODO: upload image in background")
			},
		},
		trayhost_NewSeparatorMenuItem(),
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

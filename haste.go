package main

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/fatih/color"
	"github.com/ssddanbrown/haste/engine"
	"github.com/ssddanbrown/haste/options"
	"github.com/ssddanbrown/haste/server"
)

func main() {

	// Get options and parse command line options
	opts := options.NewOptions()
	err := opts.ParseCommandFlags()
	check(err)

	// Create a new manager
	manager := engine.NewManager(opts)

	// Build all found files
	manager.BuildAll()

	// Watch if specified
	if opts.Watch {
		startWatcher(manager, opts)
	}

}

func startWatcher(m *engine.Manager, opts *options.Options) {
	ser := server.NewServer(m, opts)
	ser.AddWatchedFolder(opts.RootPath)

	color.Green(fmt.Sprintf("Server started at http://localhost:%d", opts.ServerPort))
	// TODO -> Open option? Annoying by default
	// openWebPage(fmt.Sprintf("http://localhost:%d/", ser.Port))

	err := ser.Listen()
	check(err)
}

func openWebPage(url string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		fmt.Println(url)
		err = exec.Command("xdg-open", url).Run()
	case "windows", "darwin":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Run()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

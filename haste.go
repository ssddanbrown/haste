package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fatih/color"
	"github.com/ssddanbrown/haste/engine"
	"github.com/ssddanbrown/haste/server"
)

func main() {

	watch := flag.Bool("w", false, "Watch HTML file and auto-compile")
	port := flag.Int("p", 8088, "Provide a port to listen on")
	disableLiveReload := flag.Bool("l", false, "Disable livereload (When watching only)")

	//verbose := flag.Bool("v", false, "Enable verbose output")
	distPtr := flag.String("d", "./dist/", "Output folder for generated content")
	rootPathPtr := flag.String("r", "./", "The root relative directory build path for template location")

	flag.Parse()
	args := flag.Args()

	wd, err := os.Getwd()
	rootPath, err := filepath.Abs(filepath.Join(wd, *rootPathPtr))

	// If provided with directory use that as root build path
	if len(args) == 1 && *rootPathPtr == "./" {
		stat, err := os.Stat(args[0])
		if err == nil && stat.IsDir() {
			rootPath, err = filepath.Abs(filepath.Join(wd, args[0]))
		}
	}

	// Set output path
	distPath, err := filepath.Abs(filepath.Join(wd, *distPtr))
	check(err)

	// Create a new manager
	manager := engine.NewManager(rootPath, distPath)

	// Load in files from args or build from working directory
	if len(args) > 0 {
		for _, inputPath := range args {
			absPath := filepath.Join(wd, inputPath)
			err = manager.LoadPath(absPath)
		}
	} else {
		err = manager.LoadPath(wd)
	}
	check(err)

	// Build all found files
	manager.BuildAll()

	// Watch if specified
	if *watch {
		startWatcher(manager, *port, !*disableLiveReload)
	}

}

func startWatcher(m *engine.Manager, port int, livereload bool) {
	ser := server.NewServer(m, port, livereload)
	ser.AddWatchedFolder(m.WorkingDir)

	color.Green(fmt.Sprintf("Server started at http://localhost:%d", ser.Port))
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

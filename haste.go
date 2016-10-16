package main

import (
	"flag"
	"fmt"
	"github.com/ssddanbrown/haste/engine"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var isVerbose bool

func main() {

	watch := flag.Bool("w", false, "Watch HTML file and auto-compile")
	port := flag.Int("p", 8081, "Provide a port to listen on")
	liveReload := flag.Bool("l", false, "Enable livereload (When watching only)")
	verbose := flag.Bool("v", false, "Enable verbose ouput")
	watchDepth := flag.Int("d", 2, "Child folder watch depth (When watching only)")

	flag.Parse()
	isVerbose = *verbose

	if len(flag.Args()) < 1 {
		fmt.Println("File to parse required")
		return
	}

	readFile := flag.Args()[0]
	readFilePath, err := filepath.Abs(filepath.Join("./", readFile))
	check(err)

	// Print to stdout if not watching
	if !*watch {
		givenFile, err := os.Open(readFilePath)
		defer givenFile.Close()
		check(err)
		o, err := engine.Parse(givenFile, readFilePath)
		check(err)
		fmt.Println(o)
		return
	}

	manager := &managerServer{
		watchedFile: readFilePath,
		Port:        *port,
		LiveReload:  *liveReload,
		WatchDepth:  *watchDepth,
	}

	portFree := checkPortFree(manager.Port)

	if !portFree {
		fmt.Printf("Listen port %d not available, Are you already running haste?\n", manager.Port)
		return
	}

	manager.addWatchedFolder(readFilePath)

	fmt.Sprintf("Server started at http://localhost:%d", manager.Port)
	openWebPage(fmt.Sprintf("http://localhost:%d/", manager.Port))

	err = manager.listen()
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

func devlog(s string) {
	if isVerbose {
		fmt.Println(s)
	}
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func intInSlice(integer int, list []int) bool {
	for _, v := range list {
		if v == integer {
			return true
		}
	}
	return false
}

func getGenFileName(originalName string) string {
	fileName := filepath.Base(originalName)
	fileExt := filepath.Ext(originalName)
	fileBaseName := fileName[:len(fileName)-len(fileExt)]
	return fileBaseName + ".gen" + fileExt
}

func checkPortFree(port int) bool {

	conn, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return false
	}

	conn.Close()
	return true
}

package main

import (
	"flag"
	"fmt"
	"github.com/ssddanbrown/haste/engine"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {

	watch := flag.Bool("w", false, "Watch html file and auto-compile")

	flag.Parse()

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

	port := 35729
	portFree := checkPortFree(port)

	if !portFree {
		fmt.Printf("Listen port %d not available, Are you already running haste?\n", port)
		return
	}

	manager := &managerServer{}
	fServer, err := manager.addFileServer(readFilePath)
	check(err)
	fmt.Sprintf("Server started at http://localhost:%d", port)
	fmt.Sprintf("FileServer started at http://localhost:%d", fServer.Port)
	openWebPage(fmt.Sprintf("http://localhost:%d/%s", fServer.Port, getGenFileName(readFilePath)))
	err = manager.listen(port)
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
	// fmt.Println(s)
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

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/fatih/color"
	"github.com/ssddanbrown/haste/engine"
)

var isVerbose bool

func main() {

	// watch := flag.Bool("w", false, "Watch HTML file and auto-compile")
	// port := flag.Int("p", 8081, "Provide a port to listen on")
	// liveReload := flag.Bool("l", false, "Enable livereload (When watching only)")
	verbose := flag.Bool("v", false, "Enable verbose ouput")
	distPtr := flag.String("d", "./dist/", "Output folder for generated content")
	rootPathPtr := flag.String("r", "./", "The root relative directory build path for template location")

	flag.Parse()
	isVerbose = *verbose

	wd, err := os.Getwd()
	rootPath, err := filepath.Abs(filepath.Join(wd, *rootPathPtr))
	distPath, err := filepath.Abs(filepath.Join(wd, *distPtr))
	check(err)

	manager := engine.NewManager(rootPath, distPath)

	if len(flag.Args()) > 0 {
		for _, inputPath := range flag.Args() {
			err = manager.LoadPath(inputPath)
		}
	} else {
		err = manager.LoadPath(wd)
	}

	manager.BuildFirst()

	check(err)

	// Print to stdout if not watching
	// if !*watch {
	// 	givenFile, err := os.Open(readFilePath)
	// 	defer givenFile.Close()
	// 	check(err)
	// 	o, err := engine.Parse(givenFile, readFilePath, rootPath)
	// 	check(err)
	// 	fmt.Println(o)
	// 	return
	// }

	// if *batchMode {
	// 	batchGenerate(flag.Args(), rootPath)
	// }

	// // Watch if specified
	// if *watch {
	// 	startWatcher(readFilePath, *port, *liveReload, *watchDepth)
	// }

}

func batchGenerate(input []string, rootPath string) {
	if len(input) < 2 {
		errOut("Batch mode requires specified input files and an output folder as the last parameter. For example:")
		errOut("haste page1.html page2.html dist")
	}

	files := input[:len(input)-1]
	fileCount := len(files)
	dir := input[len(input)-1]
	waitChan := make(chan bool)
	outPath, err := filepath.Abs(filepath.Join("./", dir))
	check(err)

	for _, filePath := range files {
		go func(filePath string) {
			absInPath, err := filepath.Abs(filepath.Join("./", filePath))
			check(err)
			absOutPath, err := filepath.Abs(filepath.Join("./", dir, filePath))
			check(err)

			file, err := os.Open(absInPath)
			defer file.Close()
			check(err)

			content, err := engine.Parse(file, filePath, rootPath)
			check(err)

			// Write file to ouput
			outDir := filepath.Dir(absOutPath)
			if _, fileErr := os.Stat(outDir); fileErr != nil {
				if os.IsNotExist(err) {
					os.MkdirAll(outDir, 0755)
				} else {
					check(err)
				}
			}
			outFile, err := os.Create(absOutPath)
			defer outFile.Close()
			check(err)
			outFile.WriteString(content)
			outFile.Sync()
			devlog(fmt.Sprintf("File from:\n%s\nparsed and written to:\n%s", absInPath, absOutPath))
			waitChan <- true
		}(filePath)
	}

	finCount := 0
	for finCount < fileCount {
		_ = <-waitChan
		finCount++
	}

	color.Green("%d files successfully generated into folder %s", fileCount, outPath)
}

func startWatcher(path string, port int, livereload bool, depth int) {
	manager := &managerServer{
		WatchedPath: path,
		Port:        port,
		LiveReload:  livereload,
		WatchDepth:  depth,
	}

	portFree := checkPortFree(manager.Port)
	if !portFree {
		fmt.Printf("Listen port %d not available, Are you already running haste?\n", manager.Port)
		return
	}

	manager.addWatchedFolder(path)

	color.Green(fmt.Sprintf("Server started at http://localhost:%d", manager.Port))
	openWebPage(fmt.Sprintf("http://localhost:%d/", manager.Port))

	err := manager.listen()
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
		color.Blue(s)
	}
}

func errlog(err error) {
	if err != nil {
		color.Red(err.Error())
	}
}

func errOut(m string) {
	color.Red(m)
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

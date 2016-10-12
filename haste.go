package main

import (
	"flag"
	"fmt"
	"github.com/ssddanbrown/haste/engine"
	"os"
	"path/filepath"
)

func main() {

	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Println("File to parse required")
		return
	}

	readFile := flag.Args()[0]
	readFilePath, err := filepath.Abs(filepath.Join("./", readFile))
	check(err)

	givenFile, err := os.Open(readFilePath)
	defer givenFile.Close()
	check(err)

	o, err := engine.Parse(givenFile, readFilePath)
	if err != nil {
		fmt.Println("error: " + err.Error())
	}
	fmt.Println(o)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

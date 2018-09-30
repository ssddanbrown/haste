package engine

import (
	"bytes"
	"github.com/ssddanbrown/haste/loading"
	"github.com/ssddanbrown/haste/options"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func getTempDirOptions(t *testing.T) (func(), *options.Options) {

	dir, err := ioutil.TempDir("", "haste_test")
	if err != nil {
		t.Fatalf("Recieved error while creating temp directory: %s", err)
	}

	opts := options.NewOptions()
	opts.RootPath = dir
	opts.OutPath = filepath.Join(dir, "./dist/")
	opts.TemplateResolver = loading.NewFileTemplateResolver(opts.RootPath)
	err = os.Mkdir(opts.OutPath, 0777)
	if err != nil {
		t.Fatalf("Recieved error while creating temp dist directory: %s", err)
	}

	return func() {
		os.RemoveAll(dir)
	}, opts
}

func writeTestFile(t *testing.T, name string, content string, opts *options.Options) string {
	inputFileName := filepath.Join(opts.RootPath, name)
	inputFile := bytes.TrimSpace([]byte(content))
	err := ioutil.WriteFile(inputFileName, inputFile, 0664)
	if err != nil {
		t.Fatalf("Recieved error while creating temp file (%s): %s", inputFileName, err)
	}
	return inputFileName
}

func readTestFile(t *testing.T, name string,  opts *options.Options) string {
	fileName := filepath.Join(opts.RootPath, name)
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		t.Fatalf("Recieved error while reading temp file (%s): %s", fileName, err)
	}
	return string(content)
}

func createTestDir(t *testing.T, name string, opts *options.Options) string {
	folderName := filepath.Join(opts.RootPath, name)
	err := os.MkdirAll(folderName, 0777)
	if err != nil {
		t.Fatalf("Recieved error while creating temp directory (%s): %s", folderName, err)
	}
	return folderName
}

func TestManagerCreationLoadsFilesRecursively(t *testing.T) {
	cleanup, o := getTempDirOptions(t)
	defer cleanup()

	testContent := `
<html><body></body></html>
`
	writeTestFile(t, "index.haste.html", testContent, o)
	createTestDir(t, "sub", o)
	writeTestFile(t, "sub/inner.haste.html", testContent, o)
	o.InputPaths = []string{o.RootPath}

	m := NewManager(o)
	if len(m.buildFiles) != 2 {
		t.Errorf("Expected 2 build files to exist, found %d", len(m.buildFiles))
	}

}

func TestManager_Build(t *testing.T) {
	cleanup, o := getTempDirOptions(t)
	o.InputPaths = []string{o.RootPath}
	defer cleanup()

	testContent := `
@test=cat
<html><body>{{test}}</body></html>
`
	writeTestFile(t, "index.haste.html", testContent, o)
	expectedContent := strings.TrimSpace(`
<html><body>cat</body></html>
`)

	m := NewManager(o)
	reader, err := m.Build(m.buildFiles["index.haste.html"])
	if err != nil {
		t.Fatalf("Error while running build: %s", err)
	}
	output, err := ioutil.ReadAll(reader)
	outputStr := string(output)
	if err != nil {
		t.Fatalf("Error while reading build output: %s", err)
	}

	if outputStr != expectedContent {
		t.Fatal(buildResultErrorMessage(expectedContent, outputStr))
	}
}

func TestManager_BuildToFile(t *testing.T) {
	cleanup, o := getTempDirOptions(t)
	o.InputPaths = []string{o.RootPath}
	defer cleanup()

	testContent := `
@test=cat
<html><body>{{test}}</body></html>
`
	writeTestFile(t, "index.haste.html", testContent, o)
	expectedContent := strings.TrimSpace(`
<html><body>cat</body></html>
`)

	m := NewManager(o)
	_, err := m.BuildToFile(m.buildFiles["index.haste.html"])
	if err != nil {
		t.Fatalf("Error while running build: %s", err)
	}

	expectedOutPath := filepath.Join(o.OutPath, "index.html")
	_, err = os.Stat(expectedOutPath)
	if os.IsNotExist(err) {
		t.Fatalf("Expected outfile file not found at %s", expectedOutPath)
	}

	outputStr := readTestFile(t, "dist/index.html", o)

	if outputStr != expectedContent {
		t.Fatal(buildResultErrorMessage(expectedContent, outputStr))
	}
}

func TestManager_BuildAll(t *testing.T) {
	cleanup, o := getTempDirOptions(t)
	o.InputPaths = []string{o.RootPath}
	defer cleanup()

	testContent := `
@test=cat
<html><body>{{test}}</body></html>
`
	writeTestFile(t, "index.haste.html", testContent, o)
	writeTestFile(t, "about.haste.html", testContent, o)
	expectedContent := strings.TrimSpace(`
<html><body>cat</body></html>
`)

	m := NewManager(o)
	m.BuildAll()

	outFiles := []string{"index.html", "about.html"}

	for _, outFile := range outFiles {
		expectedOutPath := filepath.Join(o.OutPath, outFile)
		_, err := os.Stat(expectedOutPath)
		if os.IsNotExist(err) {
			t.Fatalf("Expected outfile file not found at %s", expectedOutPath)
		}

		outputStr := readTestFile(t, "dist/" + outFile, o)

		if outputStr != expectedContent {
			t.Fatal(buildResultErrorMessage(expectedContent, outputStr))
		}
	}
}

func TestManager_NotifyChange(t *testing.T) {
	cleanup, o := getTempDirOptions(t)
	o.InputPaths = []string{o.RootPath}
	defer cleanup()

	testContent := `<html><body><t:include/></body></html>`

	writeTestFile(t, "index.haste.html", testContent, o)
	writeTestFile(t, "include.html", "", o)

	m := NewManager(o)
	_, err := m.BuildToFile(m.buildFiles["index.haste.html"])
	if err != nil {
		t.Fatalf("Error while running build: %s", err)
	}

	outputStr := readTestFile(t, "dist/index.html", o)
	expectedContent := "<html><body></body></html>"
	if outputStr != expectedContent {
		t.Fatal(buildResultErrorMessage(expectedContent, outputStr))
	}

	writeTestFile(t, "include.html", "<p>hello</p>", o)
	m.NotifyChange("include.html")

	outputStr = readTestFile(t, "dist/index.html", o)
	expectedContent = "<html><body><p>hello</p></body></html>"
	if outputStr != expectedContent {
		t.Fatal(buildResultErrorMessage(expectedContent, outputStr))
	}
}

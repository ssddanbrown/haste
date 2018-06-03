package engine

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// A Manager keeps control over builds and keeps track of what files are in build
// in addition to containing build-based configuration such as syntax patterns.
type Manager struct {
	WorkingDir     string
	OutDir         string
	BuildFiles     []*BuildFile
	BuildFilePaths map[string]bool

	DependMap   map[string][]*BuildFile
	Glob        string
	GlobDepth   int
	TagPrefix   []byte
	VarTagOpen  []byte
	VarTagClose []byte
}

// NewManager creates and initializes a new Manager with a set of defaults
func NewManager(workingDir string, outDir string) *Manager {
	m := &Manager{
		WorkingDir:     workingDir,
		OutDir:         outDir,
		DependMap:      make(map[string][]*BuildFile, 0),
		BuildFilePaths: make(map[string]bool),
		Glob:           "*.haste.html",
		GlobDepth:      5,
		TagPrefix:      []byte("t:"),
		VarTagOpen:     []byte("{{{"),
		VarTagClose:    []byte("}}}"),
	}
	return m
}

// LoadPath of a file or directory into the manager
// Recursively searches for matching build files if a directory given.
func (m *Manager) LoadPath(path string) error {
	absPath, err := filepath.Abs(path)
	fileStat, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	if !fileStat.IsDir() {
		m.addBuildFile(absPath)
		return err
	}

	newBuildFiles, err := m.scanNewBuildFiles(absPath)
	for _, path := range newBuildFiles {
		m.addBuildFile(path)
	}

	return err
}

func (m *Manager) BuildFirst() {
	file := m.BuildFiles[0]
	bReader, _ := m.Build(file)
	output, _ := ioutil.ReadAll(bReader)
	fmt.Println(string(output))
	// TODO
}

func (m *Manager) BuildAll() {
	// TODO
}

func (m *Manager) Build(buildFile *BuildFile) (io.Reader, error) {
	file, err := os.Open(buildFile.Path)
	builder := NewBuilder(file, m, nil)
	bReader := builder.Build()
	return bReader, err
}

func (m *Manager) addBuildFile(path string) {
	if _, ok := m.BuildFilePaths[path]; ok {
		return
	}

	newBuild := NewBuildFile(path)
	m.BuildFilePaths[newBuild.Path] = true
	m.BuildFiles = append(m.BuildFiles, newBuild)
}

func (m *Manager) scanNewBuildFiles(root string) ([]string, error) {
	fileList := []string{}
	err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if match, err := filepath.Match(m.Glob, f.Name()); match && err != nil {
			fileList = append(fileList, path)
		}
		return nil
	})
	return fileList, err
}

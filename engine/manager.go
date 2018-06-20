package engine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// A Manager keeps control over builds and keeps track of what files are in build
// in addition to containing build-based configuration such as syntax patterns.
type Manager struct {
	WorkingDir     string
	OutDir         string
	BuildFiles     []*BuildFile
	buildFilePaths map[string]bool

	glob        string
	globDepth   int
	tagPrefix   []byte
	varTagOpen  []byte
	varTagClose []byte
}

// NewManager creates and initializes a new Manager with a set of defaults
func NewManager(workingDir string, outDir string) *Manager {
	m := &Manager{
		WorkingDir:     workingDir,
		OutDir:         outDir,
		buildFilePaths: make(map[string]bool),
		glob:           "*.haste.html",
		globDepth:      5,
		tagPrefix:      []byte("t:"),
		varTagOpen:     []byte("{{"),
		varTagClose:    []byte("}}"),
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

func (m *Manager) BuildAll() []string {

	var outPaths []string

	for _, bf := range m.BuildFiles {
		outPath, err := m.BuildToFile(bf)
		outPaths = append(outPaths, outPath)
		if err != nil {
			fmt.Println(err)
		}
	}

	return outPaths
}

func (m *Manager) BuildToFile(b *BuildFile) (string, error) {
	relPath, err := filepath.Rel(m.WorkingDir, b.path)
	if err != nil {
		return "", err
	}

	outPath := strings.TrimSuffix(relPath, ".haste.html") + ".html"
	outPath = filepath.Join(m.OutDir, outPath)
	outPathDir := filepath.Dir(outPath)
	err = os.MkdirAll(outPathDir, os.ModePerm)
	if err != nil {
		return outPath, err
	}

	reader, err := m.Build(b)
	file, err := os.Create(outPath)
	if err != nil {
		return outPath, err
	}

	io.Copy(file, reader)
	return outPath, err
}

func (m *Manager) Build(buildFile *BuildFile) (io.Reader, error) {
	file, err := os.Open(buildFile.path)
	builder := NewBuilder(file, m, nil)
	bReader := builder.Build()
	return bReader, err
}

func (m *Manager) addBuildFile(path string) {
	if _, ok := m.buildFilePaths[path]; ok {
		return
	}

	newBuild := NewBuildFile(path)
	m.buildFilePaths[newBuild.path] = true
	m.BuildFiles = append(m.BuildFiles, newBuild)
}

func (m *Manager) scanNewBuildFiles(root string) ([]string, error) {
	fileList := []string{}
	err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		match, err := filepath.Match(m.glob, f.Name())
		if match && err == nil {
			fileList = append(fileList, path)
		}
		return nil
	})
	return fileList, err
}

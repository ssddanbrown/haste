package engine

import (
	"fmt"
	"github.com/ssddanbrown/haste/options"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// A Manager keeps control over builds and keeps track of what files are in build
// in addition to containing build-based configuration such as syntax patterns.
type Manager struct {
	options *options.Options

	buildFiles   map[string]*BuildFile
	glob         string
	globDepth    int
}

// NewManager creates and initializes a new Manager with a set of defaults
func NewManager(options *options.Options) *Manager {
	m := &Manager{
		options: options,
		buildFiles:   make(map[string]*BuildFile),
		glob:         "*" + options.BuildFileExtension,
		globDepth:    5,
	}

	if options.InputPaths != nil {
		m.loadPaths(options.InputPaths)
	}

	return m
}

func (m *Manager) loadPaths(paths []string) error {
	for _, path := range paths {
		err := m.loadPath(path)
		if err != nil {
			return err
		}
	}
	return nil
}


func (m *Manager) loadPath(path string) error {
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
	for _, buildFilePath := range newBuildFiles {
		m.addBuildFile(buildFilePath)
	}

	return err
}

func (m *Manager) BuildAll() []string {

	var outPaths []string

	for _, bf := range m.buildFiles {
		outPath, err := m.BuildToFile(bf)
		outPaths = append(outPaths, outPath)
		if err != nil {
			fmt.Println(err)
		}
	}

	return outPaths
}

func (m *Manager) BuildToFile(b *BuildFile) (string, error) {
	relPath, err := filepath.Rel(m.options.RootPath, b.path)
	if err != nil {
		return "", err
	}

	outPath := strings.TrimSuffix(relPath, ".haste.html") + ".html"
	outPath = filepath.Join(m.options.OutPath, outPath)
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

func (m *Manager) NotifyChange(file string) []string {
	var outPaths []string

	// If a BuildFile rebuild and exit
	match, err := filepath.Match(m.glob, filepath.Base(file))

	if match && err == nil {
		bf := m.addBuildFile(file)
		outPath, err := m.BuildToFile(bf)
		outPaths = append(outPaths, outPath)
		if err != nil {
			fmt.Println(err)
		}
		return outPaths
	}

	// Rebuild any BuildFiles that depend on this file
	for _, bf := range m.buildFiles {
		if _, ok := bf.includes[file]; ok {
			outPath, err := m.BuildToFile(bf)
			outPaths = append(outPaths, outPath)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
	return outPaths
}

func (m *Manager) Build(buildFile *BuildFile) (io.Reader, error) {
	fmt.Println("Building:", buildFile.path)
	file, err := os.Open(buildFile.path)
	builder := NewBuilder(file, m.options,nil)
	bReader := builder.Build()
	buildFile.includes = builder.FilesParsed
	return bReader, err
}

func (m *Manager) addBuildFile(path string) *BuildFile {
	if bf, ok := m.buildFiles[path]; ok {
		return bf
	}

	newBuild := NewBuildFile(path)
	m.buildFiles[newBuild.path] = newBuild
	return newBuild
}

func (m *Manager) scanNewBuildFiles(root string) ([]string, error) {
	var fileList []string
	err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		match, err := filepath.Match(m.glob, f.Name())
		if match && err == nil {
			fileList = append(fileList, path)
		}
		return nil
	})
	return fileList, err
}

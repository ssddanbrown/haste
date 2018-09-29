package options

import (
	"errors"
	"flag"
	"fmt"
	"github.com/ssddanbrown/haste/loading"
	"os"
	"path/filepath"
)

// Options hold all the haste specific options available
type Options struct {
	Verbose bool

	// Internal Services
	TemplateResolver loading.TemplateResolver

	// Manager Options
	OutPath string
	RootPath string
	InputPaths []string
	BuildFileExtension string

	// Build Options
	TagPrefix []byte
	VarTagPrefix []byte
	VarTagOpen []byte
	VarTagClose []byte

	// Server options
	Watch bool
	ServerPort int
	LiveReload bool
}

// NewOptions provides a new set of options with defaults set
func NewOptions() *Options {
	o := &Options{
		BuildFileExtension: ".haste.html",

		TagPrefix: []byte("t:"),
		VarTagPrefix: []byte("v:"),
		VarTagOpen: []byte("{{"),
		VarTagClose: []byte("}}"),

		Watch: false,
		ServerPort: 8081,
		LiveReload: true,
	}
	return o
}

func (o *Options) LoadFileResolver() {
	templateResolver := loading.NewFileTemplateResolver(o.RootPath)
	o.TemplateResolver = templateResolver
}

// ParseCommandFlags to read user-provided input from the command-line
// and update the options with what's provided.
func (o *Options) ParseCommandFlags() error {
	watch := flag.Bool("w", false, "Watch HTML file and auto-compile")
	port := flag.Int("p", 8081, "Provide a port to listen on")
	disableLiveReload := flag.Bool("l", false, "Disable livereload (When watching only)")
	verbose := flag.Bool("v", false, "Enable verbose output")
	distPtr := flag.String("d", "./dist/", "Output folder for generated content")
	rootPathPtr := flag.String("r", "./", "The root relative directory build path for template location")

	flag.Parse()

	o.Verbose = *verbose
	o.Watch = *watch
	o.ServerPort = *port
	o.LiveReload = !*disableLiveReload

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
	outPath, err := filepath.Abs(filepath.Join(wd, *distPtr))
	if err != nil {
		return err
	}

	err = createFolderIfNotExisting(outPath)
	if err != nil {
		return err
	}

	o.RootPath = rootPath
	o.OutPath = outPath

	// Find files to load from args or use working directory
	var inputPaths []string
	if len(args) > 0 {
		for _, inputPath := range args {
			absPath := filepath.Join(wd, inputPath)
			inputPaths = append(inputPaths, absPath)
		}
	} else {
		inputPaths = append(inputPaths, wd)
	}
	o.InputPaths = inputPaths

	return err
}

func createFolderIfNotExisting(folderPath string) error {
	_, err := os.Stat(folderPath)
	if !os.IsNotExist(err) {
		return nil
	}

	parentFolder := filepath.Dir(folderPath)
	info, err := os.Stat(parentFolder)
	if os.IsNotExist(err) || !info.IsDir() {
		return errors.New(fmt.Sprintf("Cannot find directory \"%s\" or it's parent directory"))
	}

	err = os.Mkdir(folderPath, 0777)
	return err
}
package server

import (
	"fmt"
	"github.com/fatih/color"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ssddanbrown/haste/engine"
	"github.com/ssddanbrown/haste/options"

	"errors"
	"github.com/howeyc/fsnotify"
	"golang.org/x/net/websocket"
	"net"
)

type Server struct {
	Manager          *engine.Manager
	WatchedFolders   []string
	fileWatcher      *fsnotify.Watcher
	changedFiles     chan string
	sockets          []*websocket.Conn
	lastFileChanges  map[string]int64
	WatchedPath      string
	WatchedRootFiles []string
	WatchDepth       int
	Options          *options.Options
}

func NewServer(manager *engine.Manager, opts *options.Options) *Server {
	s := &Server{
		Manager:         manager,
		WatchedPath:     opts.RootPath,
		WatchDepth:      5,
		lastFileChanges: make(map[string]int64),
		Options: opts,
	}

	portFree := checkPortFree(opts.ServerPort)
	if !portFree {
		check(errors.New(fmt.Sprintf("Listen port %d not available, Are you already running haste?\n", opts.ServerPort)))
		return s
	}

	return s
}

func (s *Server) verboseLog(str string) {
	if s.Options.Verbose {
		color.Blue(str)
	}
}

func (s *Server) watchingFolder() bool {
	return filepath.Ext(s.WatchedPath) == ""
}

func (s *Server) AddWatchedFolder(folder string) {
	err := s.watchFoldersToDepth(folder, s.WatchDepth)
	check(err)
	go s.handleFileChange(folder)
}

func (s *Server) Listen() error {
	s.startFileWatcher()
	handler := s.getManagerRouting()
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", s.Options.ServerPort), handler)
	return nil
}

func (s *Server) liveReloadAlertChange(file string) {
	response := livereloadChange{
		Command: "reload",
		Path:    file,
		LiveCSS: true,
	}
	for i := 0; i < len(s.sockets); i++ {
		if !s.sockets[i].IsServerConn() {
			s.sockets[i].Close()
		}

		if s.sockets[i].IsServerConn() {
			websocket.JSON.Send(s.sockets[i], response)
		}
	}
}

func (s *Server) getLastFileChange(changedFile string) int64 {
	lastChange, ok := s.lastFileChanges[changedFile]
	if !ok {
		return 0
	}
	return lastChange
}

func (s *Server) handleFileChange(changedFile string) {

	// Prevent duplicate changes
	currentTime := time.Now().UnixNano()
	if (currentTime-s.getLastFileChange(changedFile))/1000000 < 100 {
		return
	}

	// Ignore git directories
	if strings.Contains(changedFile, ".git/") {
		return
	}

	// Check if a relevant extension
	watchedExtensions := []string{".html", ".css", ".js"}
	reload := false
	for _, ext := range watchedExtensions {
		if filepath.Ext(changedFile) == ext {
			reload = true
		}
	}

	s.lastFileChanges[changedFile] = currentTime

	if !reload {
		s.changedFiles <- changedFile
		return
	}

	s.verboseLog(fmt.Sprintf("Change occurred in %s", changedFile))

	// Build and reload files
	time.AfterFunc(50 * time.Millisecond, func() {

		outFiles := s.Manager.NotifyChange(changedFile)

		time.AfterFunc(50 * time.Millisecond, func() {
			for _, file := range outFiles {
				s.changedFiles <- file
			}
		})
	})

}

func (s *Server) startFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	check(err)

	s.fileWatcher = watcher
	s.changedFiles = make(chan string)

	// Process events
	go func() {

		done := make(chan bool)

		for {
			select {
			case ev := <-watcher.Event:
				if ev.IsModify() {
					s.handleFileChange(ev.Name)
				}
			case err := <-watcher.Error:
				s.verboseLog("File Watcher Error: " + err.Error())
			}
		}

		// Hang so program doesn't exit
		<-done

		watcher.Close()
	}()

	for i := 0; i < len(s.WatchedFolders); i++ {
		err = watcher.Watch(s.WatchedFolders[i])
		s.verboseLog("Adding file watcher to " + s.WatchedFolders[i])
		check(err)
	}

	go func() {
		for f := range s.changedFiles {
			if len(s.sockets) > 0 {
				s.liveReloadAlertChange(f)
			}
		}
	}()

	return nil
}

func (s *Server) watchFoldersToDepth(folderPath string, depth int) error {

	ignoreFolders := []string{"node_modules", ".git"}

	s.watchFolder(folderPath)
	if depth == 0 {
		return nil
	}

	folderItems, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return err
	}

	for _, f := range folderItems {
		if f.IsDir() && !stringInSlice(f.Name(), ignoreFolders) {
			newFPath := filepath.Join(folderPath, f.Name())
			s.watchFoldersToDepth(newFPath, depth-1)
		}
	}

	return nil
}
func (s *Server) watchFolder(folderPath string) error {
	if !stringInSlice(folderPath, s.WatchedFolders) {
		s.WatchedFolders = append(s.WatchedFolders, folderPath)
		if s.fileWatcher == nil {
			return nil
		}
		err := s.fileWatcher.Watch(folderPath)
		s.verboseLog("Adding file watcher to " + folderPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) getManagerRouting() *http.ServeMux {

	handler := http.NewServeMux()
	customServeMux := http.NewServeMux()

	customServeMux.Handle("/", http.FileServer(http.Dir(s.Options.OutPath)))

	// Get our generated HTML file
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		htmlPath := filepath.Join(s.Options.OutPath, r.URL.Path)
		if filepath.Ext(htmlPath) == "" {
			htmlPath += "/index.html"
		}

		if fileExists(htmlPath) {
			file, err := os.Open(htmlPath)
			check(err)
			w.Header().Add("Cache-Control", "no-cache")
			w.Header().Add("Content-Type", "text/html")
			io.Copy(w, file)
			if s.Options.LiveReload {
				fmt.Fprintln(w, "\n<script src=\"/livereload.js\"></script>")
			}
		} else {
			fmt.Println(r.URL.Path)
			customServeMux.ServeHTTP(w, r)
		}

	})

	if !s.Options.LiveReload {
		return handler
	}

	// Get LiveReload Script
	handler.HandleFunc("/livereload.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(livereloadjs))
	})

	// Websocket handling
	wsHandler := s.getLivereloadWsHandler()
	handler.Handle("/livereload", websocket.Handler(wsHandler))

	return handler
}

func fileExists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

func check(err error) {
	if err != nil {
		panic(err)
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

func checkPortFree(port int) bool {

	conn, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return false
	}

	conn.Close()
	return true
}

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/howeyc/fsnotify"
	"github.com/ssddanbrown/haste/engine"
	"golang.org/x/net/websocket"
)

type managerServer struct {
	WatchedFolders   []string
	fileWatcher      *fsnotify.Watcher
	changedFiles     chan string
	sockets          []*websocket.Conn
	lastFileChange   int64
	Port             int
	WatchedPath      string
	WatchedRootFiles []string
	LiveReload       bool
	WatchDepth       int
}

func (m *managerServer) watchingFolder() bool {
	return filepath.Ext(m.WatchedPath) == ""
}

func (m *managerServer) addWatchedFolder(htmlFilePath string) {
	rootPath := filepath.Dir(htmlFilePath)
	err := m.watchFoldersToDepth(rootPath, m.WatchDepth)
	check(err)
	go m.handleFileChange(htmlFilePath)
}

func (m *managerServer) listen() error {
	m.startFileWatcher()
	handler := m.getManagerRouting()
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", m.Port), handler)
	return nil
}

func (m *managerServer) liveReloadAlertChange(file string) {
	response := livereloadChange{
		Command: "reload",
		Path:    file,
		LiveCSS: true,
	}
	for i := 0; i < len(m.sockets); i++ {
		if !m.sockets[i].IsServerConn() {
			m.sockets[i].Close()
		}

		if m.sockets[i].IsServerConn() {
			websocket.JSON.Send(m.sockets[i], response)
		}
	}

	devlog("File changed: " + file)
}

func (m *managerServer) handleFileChange(changedFile string) {

	// Prevent duplicate changes
	currentTime := time.Now().UnixNano()
	if (currentTime-m.lastFileChange)/1000000 < 100 {
		return
	}

	// Ignore git directories
	if strings.Contains(changedFile, ".git") {
		return
	}

	if strings.Contains(changedFile, ".gen.html") {
		return
	}

	// Check if a relevant extension
	watchedExtensions := []string{".html", ".css", ".js"}
	reload := false
	for i := range watchedExtensions {
		if filepath.Ext(changedFile) == watchedExtensions[i] {
			reload = true
		}
	}

	// Add file to known root folders

	if reload {
		time.AfterFunc(100*time.Millisecond, func() {
			in, err := ioutil.ReadFile(m.WatchedPath)
			check(err)

			r := strings.NewReader(string(in))

			// TODO - To change dir used
			newContent, err := engine.Parse(r, m.WatchedPath, filepath.Dir(m.WatchedPath))
			if err != nil {
				errlog(err)
				return
			}

			newFileName := getGenFileName(m.WatchedPath)
			newFileLocation := filepath.Join(filepath.Dir(m.WatchedPath), "./"+newFileName)
			newFile, err := os.Create(newFileLocation)
			defer newFile.Close()
			check(err)

			newFile.WriteString(newContent)
			newFile.Sync()
			m.changedFiles <- newFileLocation
		})

	} else {
		m.changedFiles <- changedFile
	}

	m.lastFileChange = currentTime
}

func (m *managerServer) startFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	check(err)

	m.fileWatcher = watcher
	m.changedFiles = make(chan string)

	// Process events
	go func() {

		done := make(chan bool)

		for {
			select {
			case ev := <-watcher.Event:
				if ev.IsModify() {
					m.handleFileChange(ev.Name)
				}
			case err := <-watcher.Error:
				devlog("File Watcher Error: " + err.Error())
			}
		}

		// Hang so program doesn't exit
		<-done

		watcher.Close()
	}()

	for i := 0; i < len(m.WatchedFolders); i++ {
		err = watcher.Watch(m.WatchedFolders[i])
		devlog("Adding file watcher to " + m.WatchedFolders[i])
		check(err)
	}

	go func() {
		for f := range m.changedFiles {
			if len(m.sockets) > 0 {
				m.liveReloadAlertChange(f)
			}
		}
	}()

	return nil
}

func (m *managerServer) watchFoldersToDepth(folderPath string, depth int) error {

	ignoreFolders := []string{"node_modules", ".git"}

	m.watchFolder(folderPath)
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
			m.watchFoldersToDepth(newFPath, depth-1)
		}
	}

	return nil
}
func (m *managerServer) watchFolder(folderPath string) error {
	if !stringInSlice(folderPath, m.WatchedFolders) {
		m.WatchedFolders = append(m.WatchedFolders, folderPath)
		if m.fileWatcher == nil {
			return nil
		}
		err := m.fileWatcher.Watch(folderPath)
		devlog("Adding file watcher to " + folderPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (manager *managerServer) getManagerRouting() *http.ServeMux {

	handler := http.NewServeMux()
	customServeMux := http.NewServeMux()

	customServeMux.Handle("/", http.FileServer(http.Dir(filepath.Dir(manager.WatchedPath))))

	// Get our generated HTML file
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path != "/" {
			fmt.Println(r.URL.Path)
			customServeMux.ServeHTTP(w, r)
		} else {

			file, err := os.Open(getGenFileName(manager.WatchedPath))
			check(err)
			w.Header().Add("Cache-Control", "no-cache")
			w.Header().Add("Content-Type", "text/html")
			io.Copy(w, file)
			if manager.LiveReload {
				fmt.Fprintln(w, "\n<script src=\"/livereload.js\"></script>")
			}
		}

	})

	if !manager.LiveReload {
		return handler
	}

	// Load compiled in static content
	fileBox := rice.MustFindBox("res")

	// Get LiveReload Script
	handler.HandleFunc("/livereload.js", func(w http.ResponseWriter, r *http.Request) {
		http.FileServer(fileBox.HTTPBox())
		scriptString := fileBox.MustString("livereload.js")
		templS, err := template.New("livereload").Parse(scriptString)
		if err != nil {
			check(err)
		}
		templS.Execute(w, manager.Port)
	})

	// Websocket handling
	wsHandler := manager.getLivereloadWsHandler()
	handler.Handle("/livereload", websocket.Handler(wsHandler))

	return handler
}

type livereloadResponse struct {
	Command string `json:"command"`
}

type livereloadHello struct {
	Command    string   `json:"command"`
	Protocols  []string `json:"protocols"`
	ServerName string   `json:"serverName"`
}

type livereloadChange struct {
	Command string `json:"command"`
	Path    string `json:"path"`
	LiveCSS bool   `json:"liveCSS"`
}

func (manager *managerServer) getLivereloadWsHandler() func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {

		manager.sockets = append(manager.sockets, ws)

		for {
			// websocket.Message.Send(ws, "Hello, Client!")
			wsData := new(livereloadResponse)
			err := websocket.JSON.Receive(ws, &wsData)
			if err != nil && err != io.EOF {
				check(err)
				return
			} else if err == io.EOF {
				return
			}

			if wsData.Command == "hello" {
				response := livereloadHello{
					Command: "hello",
					Protocols: []string{
						"http://livereload.com/protocols/connection-check-1",
						"http://livereload.com/protocols/official-7",
						"http://livereload.com/protocols/official-8",
						"http://livereload.com/protocols/official-9",
						"http://livereload.com/protocols/2.x-origin-version-negotiation",
					},
					ServerName: "Webby",
				}
				devlog("Sending livereload hello")
				websocket.JSON.Send(ws, response)
			}

		}

	}

}

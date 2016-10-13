package main

import (
	"fmt"
	"github.com/GeertJohan/go.rice"
	"github.com/howeyc/fsnotify"
	"github.com/ssddanbrown/haste/engine"
	"golang.org/x/net/websocket"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type managerServer struct {
	FileServers     []*fileServer
	WatchedFolders  []string
	fileWatcher     *fsnotify.Watcher
	changedFiles    chan string
	sockets         []*websocket.Conn
	lastFileChange  int64
	Port            int
	NetworkIp       string
	templatingFiles []string
}

func (m *managerServer) addFileServer(htmlFilePath string) (*fileServer, error) {
	rootPath := filepath.Dir(htmlFilePath)
	m.templatingFiles = append(m.templatingFiles, htmlFilePath)

	fServer, err := startFileServer(rootPath)
	if err != nil {
		return nil, err
	}
	m.FileServers = append(m.FileServers, fServer)

	m.watchFolder(fServer.RootPath)
	go m.handleFileChange(htmlFilePath)
	return fServer, nil
}

func (m *managerServer) listen(port int) error {
	m.startFileWatcher()
	m.Port = port
	m.NetworkIp = getLocalIp()
	handler := m.getManagerRouting()
	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), handler)
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
		// devlog("GITCHANGE")
		return
	}

	if strings.Contains(changedFile, ".gen.html") {
		return
	}

	template := false
	for i := range m.templatingFiles {
		if m.templatingFiles[i] == changedFile {
			template = true
		}
	}

	if template {
		time.AfterFunc(100*time.Millisecond, func() {
			in, err := ioutil.ReadFile(changedFile)
			check(err)

			r := strings.NewReader(string(in))

			newContent, err := engine.Parse(r, changedFile)
			if err != nil {
				devlog(err.Error())
				return
			}

			newFileName := getGenFileName(changedFile)
			newFileLocation := filepath.Join(filepath.Dir(changedFile), "./"+newFileName)
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

	// Load compiled in static content
	fileBox := rice.MustFindBox("res")

	// Get LiveReload Script
	handler.Handle("/livereload.js", http.FileServer(fileBox.HTTPBox()))

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

func getLocalIp() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipString := ipnet.IP.String()
				if strings.Index(ipString, "192.168") == 0 || strings.Index(ipString, "10.") == 0 {
					return ipString
				}
			}
		}
	}
	return ""
}

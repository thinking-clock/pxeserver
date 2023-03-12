package main

import (
	"html/template"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	_ "embed"
)

//go:embed status.html.tmpl
var statusHtml string

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type progressRecord struct {
	WOL       int
	Menu      int
	Vmlinuz   int
	Initrd    int
	Iso       int
	CloudInit int
}

func setWOLProgress(hostName string) {
	progressLock.Lock()
	progress[hostName].WOL = progressMax.WOL
	progress[hostName].Menu = 0
	progress[hostName].Vmlinuz = 0
	progress[hostName].Initrd = 0
	progress[hostName].Iso = 0
	progress[hostName].CloudInit = 0
	curtinLog[hostName] = []map[string]interface{}{}
	progressCond.Broadcast()
	progressLock.Unlock()
}

func setMenuProgress(hostName string) {
	progressLock.Lock()
	progress[hostName].Menu = progressMax.Menu
	progressCond.Broadcast()
	progressLock.Unlock()
}

func setVmlinuzProgress(hostName string) {
	progressLock.Lock()
	progress[hostName].Vmlinuz = progressMax.Vmlinuz
	progressCond.Broadcast()
	progressLock.Unlock()
}

func setInitrdProgress(hostName string) {
	progressLock.Lock()
	progress[hostName].Initrd = progressMax.Initrd
	progressCond.Broadcast()
	progressLock.Unlock()
}

func setIsoProgress(hostName string) {
	progressLock.Lock()
	progress[hostName].Iso = progressMax.Iso
	progressCond.Broadcast()
	progressLock.Unlock()
}

func setCloudInitProgress(hostName string) {
	progressLock.Lock()
	progress[hostName].CloudInit = progressMax.CloudInit
	progressCond.Broadcast()
	progressLock.Unlock()
}

func appendLog(hostName string, logMsg map[string]interface{}) {
	progressLock.Lock()
	logArr := curtinLog[hostName]
	logArr = append(logArr, logMsg)
	if len(logArr) > 10 {
		logArr = logArr[len(logArr)-10:]
	}
	curtinLog[hostName] = logArr
	progressCond.Broadcast()
	progressLock.Unlock()
}

var progressLock sync.Mutex
var progressCond *sync.Cond = sync.NewCond(&progressLock)
var progress map[string]*progressRecord = make(map[string]*progressRecord)
var curtinLog map[string][]map[string]interface{} = map[string][]map[string]interface{}{
	"log": nil,
}

var progressMax = progressRecord{
	WOL:       10,
	Menu:      10,
	Vmlinuz:   15,
	Initrd:    25,
	Iso:       30,
	CloudInit: 10,
}

type htmlFill struct {
	Host        string
	Port        string
	Inventory   map[string]string
	Progress    map[string]*progressRecord
	ProgressMax progressRecord
}

func statusPage(host string, httpPort string, inventory map[string]string) func(http.ResponseWriter, *http.Request) {
	tmpl, err := template.New("status.tmpl").Parse(statusHtml)
	if err != nil {
		log.Fatalf("Could not create template: %s", err)
	}

	for _, hostname := range inventory {
		progress[hostname] = &progressRecord{}
	}

	_, port, err := net.SplitHostPort(httpPort)
	if err != nil {
		log.Fatalf("Could not parse address: %s", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		data := &htmlFill{
			Host:        host,
			Port:        port,
			Inventory:   inventory,
			Progress:    progress,
			ProgressMax: progressMax,
		}
		err := tmpl.Execute(w, data)
		if err != nil {
			log.Errorf("Error returning user-data: %s", err)
		}
	}
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	// helpful log statement to show connections
	log.Debugln("Websocket Client Connected")

	progressLock.Lock()

	if err := ws.WriteJSON(progress); err != nil {
		progressLock.Unlock()
		log.Println(err)
		return
	}

	progressLock.Unlock()

	for {
		progressLock.Lock()
		progressCond.Wait()

		if err := ws.WriteJSON(progress); err != nil {
			progressLock.Unlock()
			log.Println(err)
			return
		}

		if err := ws.WriteJSON(curtinLog); err != nil {
			progressLock.Unlock()
			log.Println(err)
			return
		}

		progressLock.Unlock()
	}
}

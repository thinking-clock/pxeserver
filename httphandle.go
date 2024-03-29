package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	wol "github.com/ghthor/gowol"
	log "github.com/sirupsen/logrus"
)

func handleWOL(inventory map[string]string, brdAddr string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		host := r.URL.Query().Get("h")
		for macAddr, hostname := range inventory {
			if host == "" || host == hostname {
				log.Infof("Step 0: Sending WOL to %s", hostname)
				err := wol.MagicWake(macAddr, brdAddr)
				if err != nil {
					log.Fatalf("Could not send WOL: %s", err.Error())
				}

				setWOLProgress(hostname)
			}
		}
	}
}

func logStaticRequest(fs http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.Method, r.URL.String())
		fs.ServeHTTP(w, r)
		hostIP, _, _ := net.SplitHostPort(r.RemoteAddr)
		if hostname, ok := ipHostMap.Load(hostIP); ok {
			switch {
			case strings.HasSuffix(r.URL.String(), ".iso"):
				setIsoProgress(hostname.(string))
			case strings.HasSuffix(r.URL.String(), "vmlinuz"):
				setVmlinuzProgress(hostname.(string))
			case strings.HasSuffix(r.URL.String(), "initrd"):
				setInitrdProgress(hostname.(string))
			}
		}
	}
	return http.HandlerFunc(fn)
}

var logHostname = regexp.MustCompile(`/cloudlog/(?P<Hostname>[a-z0-9]+)$`)

func cloudLogHandle() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		match := logHostname.FindStringSubmatch(r.URL.Path)
		if len(match) != 2 {
			http.NotFound(w, r)
			return
		}

		jsonMap := make(map[string]interface{})
		err := json.NewDecoder(r.Body).Decode(&jsonMap)
		if err != nil {
			log.Printf("ERROR: fail to unmarshla json, %s", err.Error())
		}
		log.Infof("LOG %s: %+v", r.URL.String(), jsonMap)
		appendLog(match[1], jsonMap)
		fmt.Fprint(w, "OK\n\n")
	}
}

func favicon(w http.ResponseWriter, r *http.Request) {
	log.Infof("%s\n", r.RequestURI)
	w.Header().Set("Content-Type", "image/x-icon")
	w.Header().Set("Cache-Control", "public, max-age=7776000")
	fmt.Fprintln(w, "data:image/x-icon;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQEAYAAABPYyMiAAAABmJLR0T///////8JWPfcAAAACXBIWXMAAABIAAAASABGyWs+AAAAF0lEQVRIx2NgGAWjYBSMglEwCkbBSAcACBAAAeaR9cIAAAAASUVORK5CYII=\n")
}

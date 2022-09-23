package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	tftp "github.com/pin/tftp/v3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	//_ "net/http/pprof"
)

type templateFill struct {
	Hostname     string
	Server       string
	HttpPort     int
	PasswordHash string
}

var ipHostMap sync.Map

type serveCfg struct {
	Host    string
	Http    int
	Tftp    int
	Datadir string
}

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/pxeserver/")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("fatal error config file: %w", err)
	}

	var hostCfg serveCfg
	viper.UnmarshalKey("serve", &hostCfg)
	inventory := viper.GetStringMapString("inventory")
	userData := viper.GetString("cloudinit")
	bootMenu := viper.GetString("bootmenu")
	passwordHash := viper.GetString("passwordhash")
	dirname, err := filepath.Abs(hostCfg.Datadir)
	if err != nil {
		log.Fatalf("Could not get absolute path to directory: %s: %s", dirname, err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(2)

	log.Infof("Serving %s on HTTP port %d", dirname, hostCfg.Http)
	log.Infof("Serving %s on TFTP port %d", dirname, 69)

	go func() {
		defer wg.Done()
		fs := http.FileServer(http.Dir(dirname))
		http.Handle("/static/", logStaticRequest(http.StripPrefix("/static/", fs)))
		http.HandleFunc("/autoinstall/", cloudInitHandler(hostCfg.Host, hostCfg.Http, userData, passwordHash))
		http.HandleFunc("/magic", handleWOL(inventory))
		http.HandleFunc("/status", statusPage(hostCfg.Host, hostCfg.Http, inventory))
		http.HandleFunc("/status/ws", wsEndpoint)
		http.HandleFunc("/cloudlog/", cloudLogHandle())
		http.HandleFunc("/favicon.ico", favicon)

		err = http.ListenAndServe(fmt.Sprintf(":%d", hostCfg.Http), nil)
		if err != nil {
			log.Fatalf("Could not serve directory: %s: %s", dirname, err.Error())
		}
	}()

	go func() {
		s := tftp.NewServer(tftpReadHandler(hostCfg.Host, hostCfg.Http, dirname, inventory, bootMenu), tftpWriteHandler)
		//s.SetTimeout(60 * time.Second) // optional
		//s.SetAnticipate(10)
		s.SetBlockSize(5000)
		s.SetBackoff(func(attempts int) time.Duration {
			return time.Duration(attempts) * time.Second
		})

		defer wg.Done()
		err := s.ListenAndServe(":69")
		if err != nil {
			log.Fatalf("TFTP error: %v\n", err)
		}
	}()

	wg.Wait()
}

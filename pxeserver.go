package main

import (
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tftp "github.com/pin/tftp/v3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var ipHostMap sync.Map

type serveCfg struct {
	Host      string
	Http      string
	Https     string
	Tftp      string
	Broadcast string
	Datadir   string
}

type acmeCfg struct {
	Root  string
	CAurl string
}

func main() {
	viper.SetConfigName("pxeserver")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("pxeserver")
	viper.EnvKeyReplacer(strings.NewReplacer("_", "."))
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("fatal error config file: %w", err)
	}

	var hostCfg serveCfg
	viper.UnmarshalKey("serve", &hostCfg)
	dirname, err := filepath.Abs(hostCfg.Datadir)
	if err != nil {
		log.Fatalf("Could not get absolute path to directory: %s: %s", dirname, err)
	}

	inventory := viper.GetStringMapString("inventory")
	var acmeCfg acmeCfg
	viper.UnmarshalKey("acme", &acmeCfg)

	tlsConfig, m, err := setupACME(hostCfg.Host, &acmeCfg)
	if err != nil {
		log.Fatalf("Could not setup ACME for certificates: %s", err)
	}

	var wg sync.WaitGroup
	wg.Add(3)

	fs := logStaticRequest(http.FileServer(http.Dir(dirname)))
	go func() {
		defer wg.Done()

		httpHandler := http.NewServeMux()
		httpHandler.Handle("/", fs)
		httpHandler.HandleFunc("/cloudlog/", cloudLogHandle())

		httpServe := &http.Server{
			Addr:    hostCfg.Http,
			Handler: m.HTTPHandler(httpHandler),
		}
		err = httpServe.ListenAndServe()
		if err != nil {
			log.Fatalf("Could not serve directory: %s: %s", dirname, err.Error())
		}
	}()

	go func() {
		defer wg.Done()
		mux := http.NewServeMux()
		tlsServer := &http.Server{
			Addr:      hostCfg.Https,
			Handler:   mux,
			TLSConfig: tlsConfig,
		}
		mux.Handle("/", fs)
		mux.HandleFunc("/magic", handleWOL(inventory, hostCfg.Broadcast))
		mux.HandleFunc("/status", statusPage(hostCfg.Host, hostCfg.Https, inventory))
		mux.HandleFunc("/status/ws", wsEndpoint)
		mux.HandleFunc("/favicon.ico", favicon)

		err = tlsServer.ListenAndServeTLS("", "")
		if err != nil {
			log.Fatalf("Could not serve pxe status: %s", err.Error())
		}
	}()

	go func() {
		s := tftp.NewServer(tftpReadHandler(dirname, inventory), tftpWriteHandler)
		s.SetBlockSize(5000)
		s.SetBackoff(func(attempts int) time.Duration {
			return time.Duration(attempts) * time.Second
		})

		defer wg.Done()
		err := s.ListenAndServe(hostCfg.Tftp)
		if err != nil {
			log.Fatalf("TFTP error: %v\n", err)
		}
	}()

	log.Infof("Serving /status on HTTPS %d", hostCfg.Https)
	log.Infof("Serving %s on HTTP %d", dirname, hostCfg.Http)
	log.Infof("Serving %s on TFTP %d", dirname, hostCfg.Tftp)
	wg.Wait()
}

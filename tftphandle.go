package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	tftp "github.com/pin/tftp/v3"
	log "github.com/sirupsen/logrus"
)

func tftpWriteHandler(filename string, wt io.WriterTo) error {
	return fmt.Errorf("read-only")
}

func tftpReadHandler(host string, httpPort int, dirname string, inventory map[string]string, bootMenu string) func(string, io.ReaderFrom) error {
	tmpl, err := template.New("pxelinux.cfg.tmpl").Parse(bootMenu)
	if err != nil {
		log.Fatalf("Could not create template: %s", err)
	}

	return func(filename string, rf io.ReaderFrom) error {
		var reader io.Reader
		var err error
		biosConfig := strings.HasPrefix(filename, "bios/pxelinux.cfg/01-")
		efiConfig := strings.HasPrefix(filename, "efi64/pxelinux.cfg/01-")
		if biosConfig || efiConfig {
			macAddr := filename
			macAddr = strings.TrimPrefix(macAddr, "bios/pxelinux.cfg/01-")
			macAddr = strings.TrimPrefix(macAddr, "efi64/pxelinux.cfg/01-")
			macAddr = strings.ReplaceAll(macAddr, "-", ":")
			macAddr = strings.ToLower(macAddr)

			hostname, ok := inventory[macAddr]
			if !ok {
				log.Printf("MAC %s not in inventory", macAddr)
			}

			buffer := &bytes.Buffer{}
			data := &templateFill{
				Hostname: hostname,
				Server:   host,
				HttpPort: httpPort,
			}
			err := tmpl.Execute(buffer, data)
			if err != nil {
				log.Printf("Error returning user-data: %s", err)
			}
			reader = buffer

			raddr := rf.(tftp.OutgoingTransfer).RemoteAddr()
			ipHostMap.Store(raddr.IP.String(), hostname)
		} else {
			reader, err = os.Open(dirname + "/" + filename)
			if err != nil {
				log.Printf("%v\n", err)
				return err
			}
		}

		n, err := rf.ReadFrom(reader)
		if err != nil {
			log.Printf("%v\n", err)
			return err
		}

		raddr := rf.(tftp.OutgoingTransfer).RemoteAddr()
		hostname, ok := ipHostMap.Load(raddr.IP.String())
		if !ok {
			return nil
		}

		if biosConfig || efiConfig {
			setMenuProgress(hostname.(string))
		} else if strings.HasSuffix(filename, "vmlinuz") {
			setVmlinuzProgress(hostname.(string))
		} else if strings.HasSuffix(filename, "initrd") {
			setInitrdProgress(hostname.(string))
		}

		log.Infof("%s: %d bytes sent to %s\n", filename, n, hostname.(string))
		return nil
	}
}

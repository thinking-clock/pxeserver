package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	tftp "github.com/pin/tftp/v3"
	log "github.com/sirupsen/logrus"
)

func tftpWriteHandler(filename string, wt io.WriterTo) error {
	return fmt.Errorf("read-only")
}

func tftpReadHandler(dirname string, inventory map[string]string) func(string, io.ReaderFrom) error {
	return func(filename string, rf io.ReaderFrom) error {
		efiConfig := strings.HasPrefix(filename, "efi64/pxelinux.cfg/01-")
		if efiConfig {
			macAddr := filename
			macAddr = strings.TrimPrefix(macAddr, "efi64/pxelinux.cfg/01-")
			macAddr = strings.ReplaceAll(macAddr, "-", ":")
			macAddr = strings.ToLower(macAddr)

			hostname, ok := inventory[macAddr]
			if !ok {
				log.Warnf("MAC %s not in inventory", macAddr)
			}

			raddr := rf.(tftp.OutgoingTransfer).RemoteAddr()
			ipHostMap.Store(raddr.IP.String(), hostname)
		}
		reader, err := os.Open(dirname + "/" + filename)
		if err != nil {
			log.Warnf("%v\n", err)
			return err
		}

		n, err := rf.ReadFrom(reader)
		if err != nil {
			log.Warnf("%v\n", err)
			return err
		}

		raddr := rf.(tftp.OutgoingTransfer).RemoteAddr()
		hostname, ok := ipHostMap.Load(raddr.IP.String())
		if !ok {
			hostname = "unknown"
		}
		log.Infof("%s: %d bytes sent to %s\n", filename, n, hostname.(string))
		if !ok {
			log.Warnf("Uknown host, cant set progress\n")
			return nil
		}
		if efiConfig {
			setMenuProgress(hostname.(string))
		} else if strings.HasSuffix(filename, "vmlinuz") {
			setVmlinuzProgress(hostname.(string))
		} else if strings.HasSuffix(filename, "initrd") {
			setInitrdProgress(hostname.(string))
		}

		return nil
	}
}

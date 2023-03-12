package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

func setupACME(hostname string, acmeCfg *acmeCfg) (*tls.Config, *autocert.Manager, error) {
	certs, err := ioutil.ReadFile(acmeCfg.Root)
	if err != nil {
		log.Fatalf("Failed to append %q to RootCAs: %v", acmeCfg.Root, err)
	}

	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		log.Fatalf("Could not append cert")
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("error generating key: %v", err)
	}

	m := &autocert.Manager{
		Prompt:      autocert.AcceptTOS,
		HostPolicy:  autocert.HostWhitelist(hostname),
		RenewBefore: 8 * time.Hour,
		Client: &acme.Client{
			Key:          key,
			DirectoryURL: acmeCfg.CAurl,
			HTTPClient: &http.Client{Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: rootCAs,
				},
			}},
		},
	}

	tlsConfig := m.TLSConfig()
	tlsConfig.GetCertificate = func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		cert, err := m.GetCertificate(hello)
		if err != nil {
			log.Println(err)
		}
		return cert, err
	}

	return tlsConfig, m, nil
}

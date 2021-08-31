package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/libdns/duckdns"

	"github.com/caddyserver/certmagic"
)

func main() {
	// Load config from flags
	config, ustore, err := ParseConfig()
	if err != nil {
		log.Fatalln("loading configuration:", err.Error())
	}

	// Verify that config.BaseDir exists, is accessible and a directory
	fi, err := os.Stat(config.BaseDir)
	if err != nil {
		log.Fatalf("error while accessing %q: %s\n", config.BaseDir, err.Error())
	}

	if !fi.IsDir() {
		log.Fatalf("%q is not a directory\n", config.BaseDir)
	}

	abs, err := filepath.Abs(config.BaseDir)
	if err != nil {
		log.Fatalln("cannot determine absolute path for directory:", err.Error())
	}

	log.Println("Serving files from", abs)

	// Set up web server mux
	mux := http.NewServeMux()

	var s = &Server{
		BaseDir:             abs,
		DisallowDirectories: config.DisallowDirectoryListings,
		UserStore:           ustore,
	}

	mux.Handle("/", s)

	if config.DuckDNSSite != "" && config.DuckDNSToken != "" {
		// Set up HTTPS certificate resolver details
		certmagic.DefaultACME.Agreed = true
		certmagic.DefaultACME.Email = config.LetsEncryptEmail
		certmagic.DefaultACME.DNS01Solver = &certmagic.DNS01Solver{
			DNSProvider: &duckdns.Provider{
				APIToken: config.DuckDNSToken,
			},
		}

		certmagic.HTTPPort = 0 // Choose a random aka free port for certmagics' HTTP to HTTPS redirect
		certmagic.HTTPSPort = config.SecurePort

		log.Println("Checking in with DuckDNS")
		err = PingDuckDNS(config.DuckDNSSite, config.DuckDNSToken)
		if err != nil {
			log.Println("[Warning] Error while telling DuckDNS our IP address:", err.Error())
		}

		go func() {
			site := fmt.Sprintf("%s.duckdns.org", config.DuckDNSSite)

			log.Println("Public HTTPS server listening on port", certmagic.HTTPSPort, "- access it over the external port configured in your router on", site)
			err := certmagic.HTTPS([]string{site}, mux)
			if err != nil {
				log.Fatalln("while running HTTPS server:", err.Error())
			}
		}()
	}

	// And the normal HTTP server
	ext, err := externalIP()
	if err == nil {
		log.Printf("Local HTTP server starting on http://%s:%d", ext, config.ServerPort)
	} else {
		log.Printf("Local HTTP server starting on port %d", config.ServerPort)
	}

	err = http.ListenAndServe(":"+strconv.Itoa(config.ServerPort), mux)
	if err != nil {
		log.Fatalln("while running HTTP server:", err.Error())
	}
}

// Source: https://stackoverflow.com/a/23558495 and https://play.golang.org/p/BDt3qEQ_2H
func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

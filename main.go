package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-acme/lego/v3/providers/dns/duckdns"

	"github.com/caddyserver/certmagic"
)

func main() {
	// Load config from flags
	config, err := ParseConfig()
	if err != nil {
		log.Fatalln("loading configuration:", err.Error())
	}

	// Set up web server mux
	mux := http.NewServeMux()

	var s = &Server{
		BaseDir:             ".",
		DisallowDirectories: config.DisallowDirectoryListings,
	}

	mux.Handle("/", s)

	if config.DuckDNSSite != "" && config.DuckDNSToken != "" {
		// Set up HTTPs server
		cfg := duckdns.NewDefaultConfig()
		cfg.Token = config.DuckDNSToken

		provider, err := duckdns.NewDNSProviderConfig(cfg)
		if err != nil {
			log.Fatalln("setting up DuckDNS provider:", err.Error())
		}

		certmagic.DefaultACME.Agreed = true
		certmagic.DefaultACME.Email = config.LetsEncryptEmail
		certmagic.DefaultACME.DNSProvider = provider

		certmagic.HTTPPort = 0 // Choose a random aka free port for certmagics' HTTP to HTTPs redirect

		log.Println("Checking in with DuckDNS")
		err = PingDuckDNS(config.DuckDNSSite, config.DuckDNSToken)
		if err != nil {
			log.Println("[Warning] Error while telling DuckDNS our IP address:", err.Error())
		}

		go func() {
			site := fmt.Sprintf("%s.duckdns.org", config.DuckDNSSite)

			log.Println("HTTPs server listening on port", certmagic.HTTPSPort, "- you can access it over the external port configured in your router on", site)
			err := certmagic.HTTPS([]string{site}, mux)
			if err != nil {
				log.Fatalln("while running HTTPs server:", err.Error())
			}
		}()
	}

	// And the normal HTTP server
	log.Printf("HTTP server starting on port %d", config.ServerPort)
	err = http.ListenAndServe(":"+strconv.Itoa(config.ServerPort), mux)
	if err != nil {
		log.Fatalln("while running HTTP server:", err.Error())
	}
}

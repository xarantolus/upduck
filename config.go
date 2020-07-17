package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ServerPort                int    `json:"server_port"`
	BaseDir                   string `json:"dir"`
	DisallowDirectoryListings bool   `json:"disallow_listings"`

	DuckDNSToken     string `json:"duck_dns_token"`
	DuckDNSSite      string `json:"duck_dns_site"`
	LetsEncryptEmail string `json:"lets_encrypt_email"`
}

var (
	serverPort                = flag.Int("p", 8080, "HTTP server port")
	baseDir                   = flag.String("dir", ".", "Directory that should be served")
	disallowDirectoryListings = flag.Bool("disallow-listings", false, "Don't show directory listings")

	letsEncryptEmail = flag.String("email", "", "Email sent to LetsEncrypt for certificate registration")
	duckDNSToken     = flag.String("token", "", "The token you get from duckdns.org")
	duckDNSSite      = flag.String("site", "", "Your duckdns.org subdomain name, e.g. \"test\" for test.duckdns.org")

	save = flag.Bool("save", false, "Save the given command line arguments to a config file located in your home directory")
)

func usage() {
	fmt.Fprint(flag.CommandLine.Output(), "upduck, a simple HTTP and HTTPs file server\n\n")
	fmt.Fprintln(flag.CommandLine.Output(), "Command-line flags:")
	flag.PrintDefaults()
	fmt.Fprint(flag.CommandLine.Output(),
		strings.ReplaceAll(`
Examples:
	Start a simple HTTP server on the default port:

		upduck 

	Start a simple HTTP server on port 2020 that doesn't show directory listings:

		upduck -p 2020 -disallow-listings
	
	Serve files from a specific directory (default is working directory):

		upduck -dir path/to/dir

	Start a HTTP server and a HTTPs server:

		upduck -email your@email.com -token DuckDNSToken -site mysite

		For this one, your router must forward any incoming connection on a port of your choosing to port 443 of the device upduck runs on. 
		This external chosen port you set in the router will be after the DuckDNS URL, e.g. https://mysite.duckdns.org:525/ for port 525.
		If you're not sure about how this works, search for "port forward tutorial" and your router model/vendor.

	Start a HTTP server on a custom port and a HTTPs server:
		
		upduck -p 2020 -email your@email.com -token DuckDNSToken -site mysite

		Here, the above notice also applies - ports must be forwarded in your router.

	You can also save your configuration so you don't need to type out everything all the time. Just run it normal and add the -save flag:

		upduck -save -p 2020 -email your@email.com -token DuckDNSToken -site mysite

		This will save your current command line. The next time, you will just need to run upduck without arguments to start the same configuration.
		You can also add the -p flag without impacting your HTTPs configuration.

	Setting your own HTTPs port is currently not really possible - you can however choose the external port in your router when forwarding ports.`,
			"\t", "  "))
}

// ParseConfig parses command-line flags
func ParseConfig() (c Config, err error) {
	flag.Usage = usage
	flag.Parse()

	c = Config{
		ServerPort:                *serverPort,
		DuckDNSToken:              *duckDNSToken,
		DuckDNSSite:               *duckDNSSite,
		LetsEncryptEmail:          *letsEncryptEmail,
		DisallowDirectoryListings: *disallowDirectoryListings,
	}

	// If we should save this to the configuration file, we exit afterwards
	if *save {
		cfgPath := getConfigPath()

		log.Println("Saving configuration to", cfgPath)
		log.Println("Please note that from now any program might be able to get your Email and Token from that file.")

		os.MkdirAll(filepath.Dir(cfgPath), 0644)

		f, err := os.Create(cfgPath)
		if err != nil {
			return c, err
		}
		defer f.Close()

		enc := json.NewEncoder(f)
		enc.SetIndent("", "\t")

		err = enc.Encode(c)
		if err != nil {
			return c, err
		}

		log.Println("Successfully saved config file.")
		os.Exit(0)
	} else if *duckDNSToken == "" && *duckDNSSite == "" && *letsEncryptEmail == "" {
		cfgPath := getConfigPath()
		// If no flags *except* maybe -p have been set, we just load the config file
		f, err := os.Open(cfgPath)
		if err != nil {
			return c, err
		}
		defer f.Close()

		err = json.NewDecoder(f).Decode(&c)
		if err != nil {
			return c, err
		}

		log.Println("Loaded config file from", cfgPath)

		// Now, if -p or -dir were given, we use that value instead of the saved one
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "p" {
				c.ServerPort = *serverPort
			}
			if f.Name == "dir" {
				c.BaseDir = *baseDir
			}
			if f.Name == "disallow-listings" {
				c.DisallowDirectoryListings = *disallowDirectoryListings
			}
		})
	}

	// Warn on certain flag combinations
	if c.DuckDNSToken == "" {
		if c.DuckDNSSite == "" {
			log.Println("Not using secure DuckDNS server")
		} else {
			log.Println("Token missing for your DuckDNS site")
		}
	} else {
		if c.DuckDNSSite == "" {
			log.Println("DuckDNS site missing, you only gave the token")
		}
	}

	return
}

// getConfigPath returns the config path while respecting certain environment variables
func getConfigPath() string {
	cfgDirPath := os.Getenv("XDG_CONFIG_HOME")
	if cfgDirPath == "" {
		hdir, err := os.UserHomeDir()
		if err != nil {
			panic("cannot determine user home directory/configuration directory (" + err.Error() + ")")
		}

		if hdir != "" {
			cfgDirPath = filepath.Join(hdir, ".config")
		}
	}

	if cfgDirPath == "" {
		panic("cannot determine user configuration directory")
	}

	return filepath.Join(cfgDirPath, ".upduck.json")
}

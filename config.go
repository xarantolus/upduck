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
	SecurePort                int    `json:"secure_port"`
	BaseDir                   string `json:"dir"`
	DisallowDirectoryListings bool   `json:"disallow_listings"`

	DuckDNSToken     string `json:"duck_dns_token"`
	DuckDNSSite      string `json:"duck_dns_site"`
	LetsEncryptEmail string `json:"lets_encrypt_email"`
}

var (
	serverPort                = flag.Int("p", 8080, "HTTP server port")
	securePort                = flag.Int("sp", 443, "HTTPS server port")
	baseDir                   = flag.String("dir", ".", "Directory that should be served")
	disallowDirectoryListings = flag.Bool("disallow-listings", false, "Disable directory listings and downloads")

	letsEncryptEmail = flag.String("email", "", "Email sent to LetsEncrypt for certificate registration")
	duckDNSToken     = flag.String("token", "", "The token you get from duckdns.org")
	duckDNSSite      = flag.String("site", "", "Your duckdns.org subdomain name, e.g. \"test\" for test.duckdns.org")

	save = flag.Bool("save", false, "Save the given command line arguments to a config file located in your home directory")
)

const (
	configFileName = ".upduck.json"
	userFileName   = ".users.upduck.json"
)

func usage() {
	fmt.Fprint(flag.CommandLine.Output(), "upduck, a simple HTTP and HTTPS file server\n\n")
	fmt.Fprintln(flag.CommandLine.Output(), "Command-line flags:")
	flag.PrintDefaults()
	fmt.Fprint(flag.CommandLine.Output(),
		strings.ReplaceAll(`
Examples:
	Start a simple HTTP server on the default port:

		> upduck 

	Start a simple HTTP server on port 2020 that doesn't show directory listings:

		> upduck -p 2020 -disallow-listings

	Serve files from a specific directory (default is working directory):

		upduck -dir path/to/dir

	Start a HTTP server and a HTTPS server:

		> upduck -email your@email.com -token DuckDNSToken -site mysite

		For this one, your router must forward any incoming connection on a port of your choosing to port 443 (or the one set with the -sp option) of the device upduck runs on. 
		This external chosen port you set in the router must be put after the DuckDNS URL, e.g. https://mysite.duckdns.org:525/ for port 525.
		If you're not sure about how this works, search for "port forward tutorial" and your router model/vendor.

	Start a HTTP server and a HTTPS server on custom ports:

		> upduck -p 2020 -sp 2121 -email your@email.com -token DuckDNSToken -site mysite

		Here, the above notice also applies - ports (in this case 2121) must be forwarded in your router.

	You can also save your configuration so you don't need to type out everything all the time. Just run it normal and add the -save flag:

		> upduck -save -p 2020 -email your@email.com -token DuckDNSToken -site mysite

		This will save your current command line. The next time, you will just need to run upduck without arguments to start the same configuration.
		You can also add the -p flag without impacting your HTTPS configuration.

User configuration:
	Upduck allows protecting your files by creating user accounts from the command-line.

	Create a new user account (or replace an existing one):

		> upduck adduser <username> <password>

	Delete a user:

		> upduck deluser <username>

	Reset all user logins:

		> upduck resetusers

	If any user accounts are configured, you need to log in before accessing files.`, "\t", "  "))
}

// ParseConfig parses command-line flags
func ParseConfig() (c Config, ustore *UserStore, err error) {
	flag.Usage = usage
	flag.Parse()

	c = Config{
		ServerPort:                *serverPort,
		DuckDNSToken:              *duckDNSToken,
		DuckDNSSite:               *duckDNSSite,
		LetsEncryptEmail:          *letsEncryptEmail,
		DisallowDirectoryListings: *disallowDirectoryListings,
		BaseDir:                   *baseDir,
		SecurePort:                *securePort,
	}

	upath := getConfigPath(userFileName)

	// If the user store does not exist, we don't warn about it. It was most likely not set up
	ustore, err = loadUsers(upath)
	if err != nil && !os.IsNotExist(err) {
		log.Println("[Warning] Error while loading user configuration file:", err.Error())
	} else if err == nil && len(ustore.Users) > 0 {
		log.Println("Loaded user configuration from", upath)
	}
	err = nil // Both errors are not critical and must not be returned

	// Allow creating user profiles
	//
	// Examples:
	// Add user:
	//     upduck adduser myname mypassword
	// Remove user:
	//     upduck deluser myname
	// Remove all users:
	//     upduck resetusers
	if flag.NFlag() == 0 && flag.NArg() > 0 {
		switch strings.ToLower(flag.Arg(0)) {
		case "adduser", "useradd", "createuser", "replaceuser":
			uname, passwd := flag.Arg(1), flag.Arg(2)
			if uname == "" {
				log.Fatalln("Username must be given")
			}
			if passwd == "" {
				log.Fatalln("Password must be given")
			}

			// Add (or replace) that user in the user store
			ustore.Users[uname] = user{
				PasswordHash: hash(passwd),
			}

			err = ustore.Save()
			if err != nil {
				log.Fatalln("Error while saving user data:", err.Error())
			}
			log.Println("Successfully added user", uname)
			os.Exit(0)
		case "deluser", "rmuser", "userdel", "userrm":
			uname := flag.Arg(1)
			if uname == "" {
				log.Fatalln("Username must be given to delete it")
			}

			// Remove user from the user store
			delete(ustore.Users, uname)

			err = ustore.Save()
			if err != nil {
				log.Fatalln("Error while saving user data:", err.Error())
			}
			log.Println("Successfully removed user")
			os.Exit(0)
		case "delallusers", "rmallusers", "resetusers":
			ustore.Users = make(map[string]user)

			err = ustore.Save()
			if err != nil {
				log.Fatalln("Error while saving user data:", err.Error())
			}
			log.Println("Successfully removed user data")
			os.Exit(0)
		}
	}

	if flag.NArg() != 0 {
		log.Printf("[Warning] Ignored arguments %q\n", strings.Join(flag.Args(), " "))
	}

	// If we should save this to the configuration file, we exit afterwards
	if *save {
		cfgPath := getConfigPath(configFileName)

		log.Println("Saving configuration to", cfgPath)
		log.Println("Please note that from now on any program might be able to get your Email and Token from that file.")

		os.MkdirAll(filepath.Dir(cfgPath), 0644)

		f, err := os.Create(cfgPath)
		if err != nil {
			return c, ustore, err
		}
		defer f.Close()

		enc := json.NewEncoder(f)
		enc.SetIndent("", "\t")

		err = enc.Encode(c)
		if err != nil {
			return c, ustore, err
		}

		log.Println("Successfully saved config file.")
		os.Exit(0)
	} else if *duckDNSToken == "" && *duckDNSSite == "" && *letsEncryptEmail == "" {
		cfgPath := getConfigPath(configFileName)
		// If no flags *except* maybe -p have been set, we just load the config file
		f, err := os.Open(cfgPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return c, ustore, err
			}
			log.Println("[Warn] Config file does not yet exist. You can create one with the -save flag")
			goto breakout
		}
		defer f.Close()

		err = json.NewDecoder(f).Decode(&c)
		if err != nil {
			return c, ustore, err
		}

		log.Println("Loaded config file from", cfgPath)

		// Now, if -p, -sp, -dir or -disallow-listings were given, we use that value instead of the saved one
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "p" {
				c.ServerPort = *serverPort
			}
			if f.Name == "sp" {
				c.SecurePort = *securePort
			}
			if f.Name == "dir" {
				c.BaseDir = *baseDir
			}
			if f.Name == "disallow-listings" {
				c.DisallowDirectoryListings = *disallowDirectoryListings
			}
		})
	}

breakout:
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
func getConfigPath(fn string) string {
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

	return filepath.Join(cfgDirPath, fn)
}

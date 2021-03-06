package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Server struct {
	BaseDir             string
	DisallowDirectories bool

	*UserStore
}

// ServeHTTP implements http.Handler by wrapping Handler with error handling and authentication
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Failed login requests are not logged
	if s.UserStore.NeedAuth() {
		uname, pw, ok := r.BasicAuth()
		if !ok {
			// We need authentication
			w.Header().Set("WWW-Authenticate", `Basic realm="Upduck login"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if !s.UserStore.IsValidUser(uname, pw) {
			// Wrong Username/Password, try again
			w.Header().Set("WWW-Authenticate", `Basic realm="Upduck login"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		log.Printf("%s: %s %s from %s\n", uname, r.Method, r.URL.String(), r.RemoteAddr)
	} else {
		// Normal logging
		log.Println(r.Method, r.URL.String(), "from", r.RemoteAddr)
	}

	err := s.Handler(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error handling %s %s from %s: %s\n", r.Method, r.URL.String(), r.RemoteAddr, err.Error())
	}
}

// Handler handles all requests
func (s *Server) Handler(w http.ResponseWriter, r *http.Request) (err error) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	p := strings.TrimPrefix(r.URL.Path, "/") // e.g. "http://server:port/test.pdf" => "test.pdf"

	direct := filepath.Join(s.BaseDir, p)

	// Prevent urls that go back too far, e.g. someone trying to access "/../secret.pdf"
	relPath, err := filepath.Rel(s.BaseDir, direct)
	if err != nil {
		return
	}

	absPath := filepath.Join(s.BaseDir, relPath)

	// Now, actually check the file
	fi, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return nil
		}
		return
	}

	// Handle directory listings
	if fi.IsDir() {
		// If a directory contains index.html, we should always serve that instead
		indexFile := filepath.Join(absPath, "index.html")
		if _, err := os.Stat(indexFile); err == nil {
			http.ServeFile(w, r, indexFile)
			return nil
		}

		if s.DisallowDirectories {
			// Listings *not* allowed
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		return s.Directory(absPath, w, r)
	}

	// Handle serving files
	return s.File(absPath, w, r)
}

// File serves the given file
func (s *Server) File(filepath string, w http.ResponseWriter, r *http.Request) (err error) {
	http.ServeFile(w, r, filepath)
	return
}

// this isn't a good and correct HTML page, but it is supposed to be minimal and works in browsers
const templateText = `
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
<title>Index of {{.Name}}</title>
<style>
html, body {
	background-color: #1a1a1a;
	color: #ccc;
}
body {
	margin: 0 auto;
	text-align: center;
	font-size: 1.25em;
}  
a {
	padding: 12px;
	color: #2c2;
}
a:hover {
	color: #2f2;
}
.dl {
	font-size:0.75em;
}
.dl > a {
	padding: 0;
}
</style>

<h2>Listing {{.Name}}</h2>
{{if .ShowBack}}<p><a href="../">Go back</a></p>{{end}}
<h3>Directories</h3>
<p class="dl">You can download this directory as <a href="?format=zip">zip</a>, <a href="?format=tar">tar</a> or <a href="?format=tar.gz">tar.gz</a> file.</p> 
{{range .Dirs}}
<p><a href="{{.Name}}/">{{.Name}}</a></p>
{{end}}

{{with .Files}}
<h3>Files</h3>
{{range .}}
<p><a href="{{.Name}}">{{.Name}}</a></p>
{{end}}{{end}}
`

type dirListing struct {
	Name     string
	ShowBack bool // Show link to ".."
	Files    []os.FileInfo
	Dirs     []os.FileInfo
}

var tmpl = template.Must(template.New("dirListing").Parse(templateText))

// Directory generates a directory listing
func (s *Server) Directory(dirPath string, w http.ResponseWriter, r *http.Request) (err error) {
	var setDownloadHeaders = func(extension string, mimetype string) {
		w.Header().Set("Content-Type", mimetype)
		w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(dirPath)+"."+extension)
		w.Header().Set("Pragma", "public")
		w.Header().Set("Expires", "0")
		w.Header().Set("Cache-Control", "must-revalidate, post-check=0, pre-check=0")
		w.Header().Set("Cache-Control", "public")
	}

	switch strings.ToUpper(r.URL.Query().Get("format")) {
	case "ZIP":
		setDownloadHeaders("zip", "application/zip")
		return GenerateZIPFromDir(w, dirPath, r.Context())
	case "TAR":
		setDownloadHeaders("tar", "application/x-tar")
		return GenerateTARFromDir(w, dirPath, r.Context())
	case "TAR.GZ":
		setDownloadHeaders("tar.gz", "application/gzip")
		return GenerateTARGZFromDir(w, dirPath, r.Context())
	default:
		dir, err := os.Open(dirPath)
		if err != nil {
			return err
		}
		defer dir.Close()

		// List everything in the given directory
		infos, err := dir.Readdir(0)
		if err != nil {
			return err
		}

		var dirs, files []os.FileInfo

		// Put them in different lists
		for _, f := range infos {
			if f.IsDir() {
				dirs = append(dirs, f)
			} else {
				files = append(files, f)
			}
		}

		// Sort everything alphabetically
		sort.Slice(dirs, func(i, j int) bool {
			return dirs[i].Name() < dirs[j].Name()
		})

		sort.Slice(files, func(i, j int) bool {
			return files[i].Name() < files[j].Name()
		})

		// If we serve the main directory, we don't show the go back link
		var showBack = filepath.Clean(s.BaseDir) != filepath.Clean(dirPath)

		dirBase := filepath.Base(dirPath)
		w.Header().Set("Content-Type", "text/html")
		return tmpl.Execute(w, dirListing{
			Name:     dirBase,
			ShowBack: showBack,
			Files:    files,
			Dirs:     dirs,
		})
	}
}

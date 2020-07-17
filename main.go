package main

import (
	"log"
	"net/http"
)

func main() {
	var s = &Server{
		BaseDir:             ".",
		DisallowDirectories: false,
	}

	http.Handle("/", s)

	log.Println("Server starting on port 80")
	http.ListenAndServe(":80", nil)
}

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	server "fs-rest/lib"
)

var rootDir string
var port string

func main() {
	cwd, _ := os.Getwd()
	flag.StringVar(&rootDir, "d", cwd, "Working Directory")
	flag.StringVar(&port, "p", "8080", "listening port")
	flag.Parse()

	fmt.Println("Listening on port " + port)
	http.ListenAndServe(":"+port, server.CreateServer(rootDir))
}

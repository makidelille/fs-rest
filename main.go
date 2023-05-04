package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
)

var rootDir string
var port string

func main() {
	cwd, _ := os.Getwd()
	flag.StringVar(&rootDir, "d", cwd, "Working Directory")
	flag.StringVar(&port, "p", "8080", "listening port")
	flag.Parse()

	fmt.Println("Listening on port " + port)
	http.ListenAndServe(":"+port, createServer(rootDir))
}

type Server struct {
	*mux.Router

	root string
}

func createServer(root string) *Server {
	server := &Server{
		Router: mux.NewRouter(),
		root:   root,
	}

	fmt.Println("Mounted on root: " + root)

	server.HandleFunc("/", server.listContainerObjects()).Methods("GET")
	server.HandleFunc("/{container}", server.listContainerObjects()).Methods("GET")
	server.HandleFunc("/{container}", server.createContainer()).Methods("PUT")
	server.HandleFunc("/{container}", server.deleteContainer()).Methods("DELETE")

	server.HandleFunc("/{container}/{object}", server.getObject()).Methods("GET")
	server.HandleFunc("/{container}/{object}", server.createObject()).Methods("PUT")
	server.HandleFunc("/{container}/{object}", server.deleteObject()).Methods("DELETE")

	return server
}

func (s *Server) listContainerObjects() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		container := mux.Vars(r)["container"]
		path := path.Join(s.root, container)

		fmt.Println("listing dir:" + path)

		filesEntry, err := os.ReadDir(path)
		if err != nil {
			fmt.Println(err.Error())
			http.NotFound(w, r)
			return
		}

		files := make([]string, len(filesEntry))
		for i, file := range filesEntry {
			files[i] = file.Name()
		}

		json.NewEncoder(w).Encode(files)
		return
	}
}

func (s *Server) createContainer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		container := mux.Vars(r)["container"]
		path := path.Join(s.root, container)
		err := os.Mkdir(path, fs.ModeDir)
		if err != nil {
			fmt.Println((err.Error()))
			http.Error(w, "Conflict", http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func (s *Server) deleteContainer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		container := mux.Vars(r)["container"]
		path := path.Join(s.root, container)
		err := os.RemoveAll(path)
		if err != nil {
			fmt.Println((err.Error()))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) getObject() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		container := mux.Vars(r)["container"]
		object := mux.Vars(r)["object"]
		ext := path.Ext(object)
		path := path.Join(s.root, container, object)

		bytes, err := os.ReadFile(path)
		if err != nil {
			fmt.Println(err.Error())
			http.NotFound(w, r)
			return
		}
		w.Header().Add("Content-type", mime.TypeByExtension(ext))
		w.Write(bytes)
	}
}

func (s *Server) createObject() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		container := mux.Vars(r)["container"]
		object := mux.Vars(r)["object"]
		path := path.Join(s.root, container, object)
		bytes, _ := io.ReadAll(r.Body)
		os.WriteFile(path, bytes, fs.ModeExclusive)
		w.WriteHeader(http.StatusCreated)
	}
}

func (s *Server) deleteObject() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		container := mux.Vars(r)["container"]
		object := mux.Vars(r)["object"]
		path := path.Join(s.root, container, object)
		os.Remove(path)
	}
}

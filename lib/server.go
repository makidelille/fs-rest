package server

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
)

type ObjectMetadata struct {
	name          string
	hash          uint64
	contentType   string
	contentLength int

	extra map[string]string
}

type ContainerMetadata struct {
	files map[string]*ObjectMetadata
}

type Server struct {
	*mux.Router
	root string

	containers map[string]*ContainerMetadata
}

func CreateServer(root string) *Server {
	server := &Server{
		Router: mux.NewRouter(),
		root:   root,

		containers: *explore(root),
	}

	fmt.Println("Mounted on root: " + root)

	server.HandleFunc("/", server.listContainerObjects(false)).Methods("GET")
	server.HandleFunc("/{container}", server.listContainerObjects(false)).Methods("GET")
	server.HandleFunc("/{container}", server.createContainer()).Methods("PUT")
	server.HandleFunc("/{container}", server.deleteContainer()).Methods("DELETE")

	server.HandleFunc("/{container}/{object}", server.getObject()).Methods("GET")
	server.HandleFunc("/{container}/{object}", server.createObject()).Methods("PUT")
	server.HandleFunc("/{container}/{object}", server.deleteObject()).Methods("DELETE")

	return server
}

func explore(root string) *map[string]*ContainerMetadata {
	record := make(map[string]*ContainerMetadata)
	dirs, _ := os.ReadDir(root)
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		record[dir.Name()] = &ContainerMetadata{
			files: make(map[string]*ObjectMetadata),
		}

		files, _ := os.ReadDir(path.Join(root, dir.Name()))

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			fmt.Println("Scaning " + dir.Name() + "/" + file.Name())
			ext := path.Ext(file.Name())
			bytes, _ := os.ReadFile(file.Name())

			meta := ObjectMetadata{
				name:          file.Name(),
				contentLength: len(bytes),
				hash:          calculateHash(bytes),
				contentType:   mime.TypeByExtension(ext),
			}

			// val, _ := json.MarshalIndent(meta, "", "   ")
			// fmt.Println(string(val))

			record[dir.Name()].files[file.Name()] = &meta

		}

	}

	// val, _ := json.MarshalIndent(record, "", "   ")
	// fmt.Println(string(val))
	return &record

}

func (s *Server) listContainerObjects(exportMeta bool) http.HandlerFunc {
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

		if exportMeta {
			json.NewEncoder(w).Encode(s.containers[container])
			return
		}

		files := make([]string, len(filesEntry))
		for i, file := range filesEntry {
			files[i] = file.Name()
		}

		json.NewEncoder(w).Encode(files)
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

		s.containers[container] = &ContainerMetadata{
			files: make(map[string]*ObjectMetadata),
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
		delete(s.containers, container)
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
		ext := path.Ext(object)
		path := path.Join(s.root, container, object)
		bytes, _ := io.ReadAll(r.Body)
		os.WriteFile(path, bytes, fs.ModeExclusive)
		w.WriteHeader(http.StatusCreated)

		s.containers[container].files[object] = &ObjectMetadata{
			name:          object,
			hash:          calculateHash(bytes),
			contentLength: len(bytes),
			contentType:   mime.TypeByExtension(ext),

			extra: make(map[string]string),
		}
	}
}

func (s *Server) deleteObject() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		container := mux.Vars(r)["container"]
		object := mux.Vars(r)["object"]
		path := path.Join(s.root, container, object)
		os.Remove(path)

		delete(s.containers[container].files, object)
	}
}

func calculateHash(b []byte) uint64 {
	h := fnv.New64a()
	h.Write(b)
	return h.Sum64()
}

package web

import (
	"io/fs"
	"net/http"
	"os"
)

func SPAHandler(root string) http.Handler {
	fileSystem := os.DirFS(root)
	if _, err := fs.Stat(fileSystem, "."); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "frontend build not found", http.StatusServiceUnavailable)
		})
	}

	files := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			files.ServeHTTP(w, r)
			return
		}
		if _, err := fs.Stat(fileSystem, r.URL.Path[1:]); err == nil {
			files.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, root+"/index.html")
	})
}

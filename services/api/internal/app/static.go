package app

import (
	"embed"
	"io/fs"
	"net/http"
)

// DistFS embeds the build directory of the web frontend.
// The taskfile will move 'web/dist' into 'services/api/internal/app/dist' before building.
//go:embed all:dist
var embeddedFiles embed.FS

// StaticHandler returns a handler that serves the frontend files.
func StaticHandler() http.Handler {
	// Try to get the 'dist' folder from the embedded FS
	dist, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		// Fallback: If dist isn't embedded (e.g. dev mode), serve nothing or a placeholder
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}

	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For a Single Page App (SPA), we want to serve index.html 
		// if the requested file doesn't exist.
		f, err := dist.Open(r.URL.Path[1:])
		if err != nil {
			// File doesn't exist, serve index.html
			r.URL.Path = "/"
		} else {
			f.Close()
		}
		
		fileServer.ServeHTTP(w, r)
	})
}

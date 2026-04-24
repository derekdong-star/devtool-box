package handler

import (
	"log"
	"net/http"

	"devtoolbox/internal/web"
)

type PageHandler struct {
	fs http.Handler
}

func NewPageHandler() *PageHandler {
	return &PageHandler{
		fs: http.FileServer(http.FS(web.FS)),
	}
}

func (h *PageHandler) Register(mux *http.ServeMux) {
	mux.Handle("/static/", h.fs)
	mux.HandleFunc("/", h.Index)
}

func (h *PageHandler) Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	b, err := web.FS.ReadFile("index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(b); err != nil {
		log.Printf("failed to write index.html response: %v", err)
	}
}

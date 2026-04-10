package handlers

import (
	"net/http"
	"strings"

	"folio/internal/store"
)

// PageThumbnailHandler serves GET /page-thumbnails/{bookID}/{pageHash}.
// It reads the pre-generated JPEG from the DB. Returns 404 when no thumbnail
// exists so the template can fall back to a placeholder.
type PageThumbnailHandler struct {
	Store *store.Store
}

func (h *PageThumbnailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/page-thumbnails/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	bookID, pageHash := parts[0], parts[1]

	data, err := h.Store.GetPageThumbnail(bookID, pageHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if data == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(data)
}

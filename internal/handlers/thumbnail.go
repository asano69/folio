package handlers

import (
	"net/http"
	"strings"

	"folio/internal/store"
)

// ThumbnailHandler serves GET /thumbnails/{bookID}.
// It reads the pre-generated JPEG from the DB. If no thumbnail exists it
// returns a 404 so the template can fall back to a placeholder.
type ThumbnailHandler struct {
	Store *store.Store
}

func (h *ThumbnailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bookID := strings.TrimPrefix(r.URL.Path, "/thumbnails/")
	bookID = strings.Trim(bookID, "/")

	if bookID == "" {
		http.Error(w, "book ID required", http.StatusBadRequest)
		return
	}

	data, err := h.Store.GetThumbnail(bookID)
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

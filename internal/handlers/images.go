package handlers

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"folio/internal/storage"
	"folio/internal/store"
)

type ImageHandler struct {
	Store *store.Store
}

// ServeHTTP handles /images/{bookID}/{filename}
func (h *ImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Strip /images/ prefix, leaving "{bookID}/{filename}"
	trimmed := strings.TrimPrefix(r.URL.Path, "/images/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "invalid image path", http.StatusBadRequest)
		return
	}
	bookID, filename := parts[0], parts[1]

	// Basic path safety check.
	if strings.Contains(filename, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		http.Error(w, "invalid file type", http.StatusBadRequest)
		return
	}

	book, err := h.Store.GetBook(bookID)
	if err != nil || book == nil {
		http.NotFound(w, r)
		return
	}

	rc, err := storage.OpenPage(book.Source, filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()

	switch ext {
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	default:
		w.Header().Set("Content-Type", "image/jpeg")
	}

	io.Copy(w, rc)
}

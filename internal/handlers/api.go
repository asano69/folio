package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"folio/internal/storage"
	"folio/internal/store"
)

// APIHandler handles REST API requests under /api/books/.
//
// Routes:
//
//	PUT  /api/books/{id}           — rename a book
//	POST /api/books/{id}/thumbnail — regenerate thumbnail
type APIHandler struct {
	Store *store.Store
}

func (h *APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/books/")

	// POST /api/books/{id}/thumbnail
	if strings.HasSuffix(path, "/thumbnail") {
		bookID := strings.TrimSuffix(path, "/thumbnail")
		if bookID == "" || strings.Contains(bookID, "/") {
			http.Error(w, "invalid book ID", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.regenerateThumbnail(w, r, bookID)
		return
	}

	// PUT /api/books/{id}
	bookID := strings.Trim(path, "/")
	if bookID == "" || strings.Contains(bookID, "/") {
		http.Error(w, "invalid book ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.renameBook(w, r, bookID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *APIHandler) renameBook(w http.ResponseWriter, r *http.Request, bookID string) {
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(body.Title)
	if title == "" {
		http.Error(w, "title cannot be empty", http.StatusBadRequest)
		return
	}

	book, err := h.Store.GetBook(bookID)
	if err != nil || book == nil {
		http.NotFound(w, r)
		return
	}

	// Update folio.json inside the CBZ first; if this fails the DB is not touched.
	if err := storage.UpdateTitle(book.Source, title); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.Store.UpdateBookTitle(bookID, title); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}{ID: bookID, Title: title})
}

// regenerateThumbnail handles POST /api/books/{id}/thumbnail.
// Generating thumbnails via an API endpoint allows future web UI integration.
func (h *APIHandler) regenerateThumbnail(w http.ResponseWriter, r *http.Request, bookID string) {
	book, err := h.Store.GetBook(bookID)
	if err != nil || book == nil {
		http.NotFound(w, r)
		return
	}

	data, err := storage.GenerateThumbnail(book.Source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.Store.UpsertThumbnail(bookID, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

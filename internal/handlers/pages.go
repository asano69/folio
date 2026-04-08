package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"folio/internal/store"
)

// PagesAPIHandler handles PUT /api/pages/{bookID}/{pageHash}.
type PagesAPIHandler struct {
	Store *store.Store
}

func (h *PagesAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/pages/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	bookID := parts[0]
	pageHash := parts[1]

	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Title     string `json:"title"`
		Attribute string `json:"attribute"`
		Body      string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	book, err := h.Store.GetBook(bookID)
	if err != nil || book == nil {
		http.NotFound(w, r)
		return
	}

	note := store.Note{
		BookID:    bookID,
		PageHash:  pageHash,
		Title:     strings.TrimSpace(body.Title),
		Attribute: body.Attribute,
		Body:      body.Body,
	}

	if err := h.Store.UpsertNote(note); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

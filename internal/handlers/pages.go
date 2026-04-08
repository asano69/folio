package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"folio/internal/store"
)

// PagesAPIHandler handles PUT /api/pages/{bookID}/{pageNumber}.
type PagesAPIHandler struct {
	Store *store.Store
}

func (h *PagesAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/pages/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	bookID := parts[0]
	pageNumber, err := strconv.Atoi(parts[1])
	if err != nil || pageNumber < 1 {
		http.Error(w, "invalid page number", http.StatusBadRequest)
		return
	}

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
		BookID:     bookID,
		PageNumber: pageNumber,
		Title:      strings.TrimSpace(body.Title),
		Attribute:  body.Attribute,
		Body:       body.Body,
	}

	if err := h.Store.UpsertNote(note); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

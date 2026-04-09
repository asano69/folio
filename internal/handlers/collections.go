package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"folio/internal/store"
)

// CollectionsAPIHandler handles REST API requests for collections.
//
// Routes:
//
//	POST   /api/collections/                        — create a collection
//	PUT    /api/collections/{id}                    — rename
//	DELETE /api/collections/{id}                    — delete (removes memberships too)
//	POST   /api/collections/{id}/books/{bookID}     — add a book
//	DELETE /api/collections/{id}/books/{bookID}     — remove a book
type CollectionsAPIHandler struct {
	Store *store.Store
}

func (h *CollectionsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/collections")
	path = strings.Trim(path, "/")

	// POST /api/collections — create
	if path == "" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.createCollection(w, r)
		return
	}

	parts := strings.SplitN(path, "/", 3)
	collectionID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "invalid collection ID", http.StatusBadRequest)
		return
	}

	// /api/collections/{id} — rename or delete
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodPut:
			h.renameCollection(w, r, collectionID)
		case http.MethodDelete:
			h.deleteCollection(w, r, collectionID)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/collections/{id}/books/{bookID} — add or remove book
	if len(parts) == 3 && parts[1] == "books" {
		bookID := parts[2]
		switch r.Method {
		case http.MethodPost:
			h.addBook(w, r, collectionID, bookID)
		case http.MethodDelete:
			h.removeBook(w, r, collectionID, bookID)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func (h *CollectionsAPIHandler) createCollection(w http.ResponseWriter, r *http.Request) {
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

	id, err := h.Store.CreateCollection(title)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
	}{ID: id, Title: title})
}

func (h *CollectionsAPIHandler) renameCollection(w http.ResponseWriter, r *http.Request, id int) {
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

	if err := h.Store.RenameCollection(id, title); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}{ID: id, Title: title})
}

func (h *CollectionsAPIHandler) deleteCollection(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.Store.DeleteCollection(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CollectionsAPIHandler) addBook(w http.ResponseWriter, _ *http.Request, collectionID int, bookID string) {
	if err := h.Store.AddBookToCollection(collectionID, bookID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CollectionsAPIHandler) removeBook(w http.ResponseWriter, _ *http.Request, collectionID int, bookID string) {
	if err := h.Store.RemoveBookFromCollection(collectionID, bookID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

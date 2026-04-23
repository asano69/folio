package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"folio/internal/store"
)

// CollectionsAPIHandler handles REST API requests for book collections.
//
// Routes:
//
//	POST   /api/collections/                        — create a collection
//	PUT    /api/collections/{id}                    — rename
//	DELETE /api/collections/{id}                    — delete
//	PUT    /api/collections/{id}/library            — move to a different library
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
	collectionID := parts[0]

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
		Name      string `json:"name"`
		LibraryID string `json:"library_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		http.Error(w, "name cannot be empty", http.StatusBadRequest)
		return
	}

	libraryID := body.LibraryID
	if libraryID == "" {
		libraryID = store.CentralLibraryID
	}

	id, err := h.Store.CreateBookCollection(name, libraryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: id, Name: name})
}

func (h *CollectionsAPIHandler) renameCollection(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		http.Error(w, "name cannot be empty", http.StatusBadRequest)
		return
	}

	if err := h.Store.RenameBookCollection(id, name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{ID: id, Name: name})
}

func (h *CollectionsAPIHandler) deleteCollection(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.Store.DeleteBookCollection(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CollectionsAPIHandler) addBook(w http.ResponseWriter, _ *http.Request, collectionID string, bookID string) {
	added, err := h.Store.AddBookToBookCollection(collectionID, bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Added bool `json:"added"`
	}{Added: added})
}

func (h *CollectionsAPIHandler) removeBook(w http.ResponseWriter, _ *http.Request, collectionID string, bookID string) {
	if err := h.Store.RemoveBookFromBookCollection(collectionID, bookID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

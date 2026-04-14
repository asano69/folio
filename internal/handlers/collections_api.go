package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"folio/internal/store"
)

// CollectionsAPIHandler handles REST API requests for book collections.
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

	id, err := h.Store.CreateBookCollection(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}{ID: id, Name: name})
}

func (h *CollectionsAPIHandler) renameCollection(w http.ResponseWriter, r *http.Request, id int) {
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
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{ID: id, Name: name})
}

func (h *CollectionsAPIHandler) deleteCollection(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.Store.DeleteBookCollection(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CollectionsAPIHandler) addBook(w http.ResponseWriter, _ *http.Request, collectionID int, bookID string) {
	added, err := h.Store.AddBookToBookCollection(collectionID, bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Return whether the book was newly added so the client can decide
	// whether to increment the displayed count.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(struct {
		Added bool `json:"added"`
	}{Added: added})
}

func (h *CollectionsAPIHandler) removeBook(w http.ResponseWriter, _ *http.Request, collectionID int, bookID string) {
	if err := h.Store.RemoveBookFromBookCollection(collectionID, bookID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SectionsAPIHandler handles REST API requests for book sections.
//
// Routes:
//
//	POST   /api/sections/     — create a section
//	PUT    /api/sections/{id} — update a section
//	DELETE /api/sections/{id} — delete a section
type SectionsAPIHandler struct {
	Store *store.Store
}

func (h *SectionsAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/sections")
	path = strings.Trim(path, "/")

	// POST /api/sections/ — create
	if path == "" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.createSection(w, r)
		return
	}

	id, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "invalid section ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.updateSection(w, r, id)
	case http.MethodDelete:
		h.deleteSection(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SectionsAPIHandler) createSection(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BookID      string `json:"book_id"`
		StartPageID int    `json:"start_page_id"`
		EndPageID   *int   `json:"end_page_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.BookID == "" {
		http.Error(w, "book_id is required", http.StatusBadRequest)
		return
	}
	if body.StartPageID == 0 {
		http.Error(w, "start_page_id is required", http.StatusBadRequest)
		return
	}

	// Verify start_page_id belongs to the given book.
	startPage, err := h.Store.GetPage(body.StartPageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if startPage == nil || startPage.BookID != body.BookID {
		http.Error(w, "start_page_id does not belong to book", http.StatusBadRequest)
		return
	}

	// Verify end_page_id belongs to the same book when provided.
	if body.EndPageID != nil {
		endPage, err := h.Store.GetPage(*body.EndPageID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if endPage == nil || endPage.BookID != body.BookID {
			http.Error(w, "end_page_id does not belong to book", http.StatusBadRequest)
			return
		}
	}

	id, err := h.Store.CreateSection(
		body.BookID,
		body.StartPageID,
		body.EndPageID,
		strings.TrimSpace(body.Title),
		body.Description,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(struct {
		ID int64 `json:"id"`
	}{ID: id})
}

func (h *SectionsAPIHandler) updateSection(w http.ResponseWriter, r *http.Request, id int) {
	var body struct {
		StartPageID int    `json:"start_page_id"`
		EndPageID   *int   `json:"end_page_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if body.StartPageID == 0 {
		http.Error(w, "start_page_id is required", http.StatusBadRequest)
		return
	}
	if !validStatuses[body.Status] {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	if err := h.Store.UpdateSection(
		id,
		body.StartPageID,
		body.EndPageID,
		strings.TrimSpace(body.Title),
		body.Description,
		body.Status,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SectionsAPIHandler) deleteSection(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.Store.DeleteSection(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

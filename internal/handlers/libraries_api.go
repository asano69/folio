package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"folio/internal/store"
)

// LibrariesAPIHandler handles REST API requests for libraries.
//
// Routes:
//
//	POST   /api/libraries/                              — create a library
//	PUT    /api/libraries/{id}                          — rename a library
//	DELETE /api/libraries/{id}                          — delete a library (fails if it has collections)
//	POST   /api/libraries/{id}/collections/{collID}     — add a collection to a library
//	DELETE /api/libraries/{id}/collections/{collID}     — remove a collection from a library
type LibrariesAPIHandler struct {
	Store *store.Store
}

func (h *LibrariesAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/libraries")
	path = strings.Trim(path, "/")

	// POST /api/libraries/ — create
	if path == "" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.createLibrary(w, r)
		return
	}

	parts := strings.SplitN(path, "/", 3)
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "invalid library ID", http.StatusBadRequest)
		return
	}

	// /api/libraries/{id}/collections/{collectionID} — add or remove collection
	if len(parts) == 3 && parts[1] == "collections" {
		collectionID, err := strconv.Atoi(parts[2])
		if err != nil {
			http.Error(w, "invalid collection ID", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodPost:
			h.addCollection(w, r, id, collectionID)
		case http.MethodDelete:
			h.removeCollection(w, r, id, collectionID)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/libraries/{id} — rename or delete
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodPut:
			h.renameLibrary(w, r, id)
		case http.MethodDelete:
			h.deleteLibrary(w, r, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func (h *LibrariesAPIHandler) createLibrary(w http.ResponseWriter, r *http.Request) {
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

	id, err := h.Store.CreateLibrary(name)
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

func (h *LibrariesAPIHandler) renameLibrary(w http.ResponseWriter, r *http.Request, id int) {
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

	if err := h.Store.RenameLibrary(id, name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{ID: id, Name: name})
}

func (h *LibrariesAPIHandler) deleteLibrary(w http.ResponseWriter, r *http.Request, id int) {
	if err := h.Store.DeleteLibrary(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// addCollection handles POST /api/libraries/{id}/collections/{collectionID}.
func (h *LibrariesAPIHandler) addCollection(w http.ResponseWriter, r *http.Request, libraryID, collectionID int) {
	added, err := h.Store.AddCollectionToLibrary(libraryID, collectionID)
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

// removeCollection handles DELETE /api/libraries/{id}/collections/{collectionID}.
func (h *LibrariesAPIHandler) removeCollection(w http.ResponseWriter, r *http.Request, libraryID, collectionID int) {
	if err := h.Store.RemoveCollectionFromLibrary(libraryID, collectionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

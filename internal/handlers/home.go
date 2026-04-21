package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"folio/internal/storage"
	"folio/internal/store"
)

// bookView is the template model for a single book card.
// Shared by HomeHandler, CollectionPageHandler, and UncategorizedPageHandler.
type bookView struct {
	ID           string
	Title        string
	HasThumbnail bool
	MissingSince string // empty means present; non-empty is the missing-since timestamp
}

// HomeHandler serves GET / — the all-books library page.
// It accepts a ?lib= query parameter to filter collections and books by library.
// Defaults to Central Library when the parameter is absent or invalid.
type HomeHandler struct {
	Store     *store.Store
	CachePath string
	Template  *template.Template
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Determine active library from ?lib= query parameter.
	activeLibraryID := store.CentralLibraryID
	if libStr := r.URL.Query().Get("lib"); libStr != "" {
		if id, err := strconv.Atoi(libStr); err == nil && id > 0 {
			activeLibraryID = id
		}
	}

	libraries, err := h.Store.ListLibraries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Verify the requested library exists; fall back to Central Library if not.
	validLibrary := false
	for _, lib := range libraries {
		if lib.ID == activeLibraryID {
			validLibrary = true
			break
		}
	}
	if !validLibrary {
		activeLibraryID = store.CentralLibraryID
	}

	// Load collections filtered to the active library for the sidebar.
	collections, err := h.Store.ListBookCollectionsInLibrary(activeLibraryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Load books for the active library.
	dbBooks, err := h.Store.ListAllBooksInLibrary(activeLibraryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalCount, err := h.Store.CountAllBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	uncategorizedCount, err := h.Store.CountUncategorizedBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Read the cache directory once to avoid N individual stat calls.
	thumbnailSet, err := storage.ListBookThumbnailIDs(h.CachePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var present, missing []bookView
	for _, b := range dbBooks {
		view := bookView{
			ID:           b.ID,
			Title:        b.Title,
			HasThumbnail: thumbnailSet[b.ID],
		}
		if b.MissingSince != nil {
			view.MissingSince = *b.MissingSince
			missing = append(missing, view)
		} else {
			present = append(present, view)
		}
	}

	data := struct {
		Books               []bookView
		MissingBooks        []bookView
		Collections         []store.BookCollection
		Libraries           []store.Library
		ActiveLibraryID     int
		ActiveCollectionID  int
		TotalBookCount      int
		UncategorizedCount  int
		IsUncategorizedPage bool
	}{
		Books:               present,
		MissingBooks:        missing,
		Collections:         collections,
		Libraries:           libraries,
		ActiveLibraryID:     activeLibraryID,
		ActiveCollectionID:  0,
		TotalBookCount:      totalCount,
		UncategorizedCount:  uncategorizedCount,
		IsUncategorizedPage: false,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

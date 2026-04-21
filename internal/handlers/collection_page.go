package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"folio/internal/storage"
	"folio/internal/store"
)

// CollectionPageHandler serves GET /collections/{id} — a single book
// collection's book list.
type CollectionPageHandler struct {
	Store     *store.Store
	CachePath string
	Template  *template.Template
}

func (h *CollectionPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/collections/")
	idStr = strings.Trim(idStr, "/")

	// /collections/ with no ID redirects to home.
	if idStr == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	collectionID, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Load the specific collection to determine its library.
	activeCollection, err := h.Store.GetBookCollection(collectionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if activeCollection == nil {
		http.NotFound(w, r)
		return
	}

	activeLibraryID := activeCollection.LibraryID

	libraries, err := h.Store.ListLibraries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Load collections for this library to populate the sidebar.
	collections, err := h.Store.ListBookCollectionsInLibrary(activeLibraryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dbBooks, err := h.Store.ListBooksInBookCollection(collectionID)
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
		ActiveCollectionID  int
		ActiveLibraryID     int
		Collection          *store.BookCollection
		TotalBookCount      int
		UncategorizedCount  int
		IsUncategorizedPage bool
	}{
		Books:               present,
		MissingBooks:        missing,
		Collections:         collections,
		Libraries:           libraries,
		ActiveCollectionID:  collectionID,
		ActiveLibraryID:     activeLibraryID,
		Collection:          activeCollection,
		TotalBookCount:      totalCount,
		UncategorizedCount:  uncategorizedCount,
		IsUncategorizedPage: false,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

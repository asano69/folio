package handlers

import (
	"html/template"
	"net/http"

	"folio/internal/storage"
	"folio/internal/store"
)

// bookView is the template model for a single book card.
// Shared by AllBooksHandler, CollectionPageHandler, and UncategorizedPageHandler.
type bookView struct {
	ID           string
	Title        string
	HasThumbnail bool
	MissingSince string // empty means present; non-empty is the missing-since timestamp
}

// shelfPageData is the common template model for all book-listing pages.
type shelfPageData struct {
	PageTitle          string
	Books              []bookView
	MissingBooks       []bookView
	EmptyMessage       string
	Collections        []store.BookCollection // all collections, for the sidebar
	ActiveCollectionID string                 // "all", "uncategorized", or UUID
	CollectionID       string                 // non-empty on collection pages (remove-from-collection button)
	TotalBookCount     int
	UncategorizedCount int
	Libraries          []store.Library // for sidebar library switcher
	ActiveLibraryID    string
}

// AllBooksHandler serves GET /collections/all — all books regardless of collection.
type AllBooksHandler struct {
	Store     *store.Store
	CachePath string
	Template  *template.Template
}

func (h *AllBooksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/collections/all" {
		http.NotFound(w, r)
		return
	}

	libID := r.URL.Query().Get("lib")
	if libID == "" {
		libID = store.CentralLibraryID
	}

	libraries, err := h.Store.ListLibraries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	collections, err := h.Store.ListBookCollectionsInLibrary(libID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dbBooks, err := h.Store.ListBooks()
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

	data := shelfPageData{
		PageTitle:          "All Books",
		Books:              present,
		MissingBooks:       missing,
		EmptyMessage:       "No books in your library yet.",
		Collections:        collections,
		ActiveCollectionID: "all",
		TotalBookCount:     totalCount,
		UncategorizedCount: uncategorizedCount,
		Libraries:          libraries,
		ActiveLibraryID:    libID,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

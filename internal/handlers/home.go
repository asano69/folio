package handlers

import (
	"html/template"
	"net/http"

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

	collections, err := h.Store.ListBookCollections()
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
		ActiveCollectionID  int
		TotalBookCount      int
		UncategorizedCount  int
		IsUncategorizedPage bool
	}{
		Books:               present,
		MissingBooks:        missing,
		Collections:         collections,
		ActiveCollectionID:  0,
		TotalBookCount:      totalCount,
		UncategorizedCount:  uncategorizedCount,
		IsUncategorizedPage: false,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

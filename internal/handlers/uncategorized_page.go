package handlers

import (
	"html/template"
	"net/http"

	"folio/internal/storage"
	"folio/internal/store"
)

// UncategorizedPageHandler serves GET /books/uncategorized — books that do not
// belong to any book collection. Uncategorized books are always presented under
// Central Library context.
type UncategorizedPageHandler struct {
	Store     *store.Store
	CachePath string
	Template  *template.Template
}

func (h *UncategorizedPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	libraries, err := h.Store.ListLibraries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Uncategorized always lives under Central Library.
	collections, err := h.Store.ListBookCollectionsInLibrary(store.CentralLibraryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dbBooks, err := h.Store.ListUncategorizedBooks()
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

	var books []bookView
	for _, b := range dbBooks {
		books = append(books, bookView{
			ID:           b.ID,
			Title:        b.Title,
			HasThumbnail: thumbnailSet[b.ID],
		})
	}

	data := struct {
		Books               []bookView
		MissingBooks        []bookView
		Collections         []store.BookCollection
		Libraries           []store.Library
		ActiveCollectionID  int
		ActiveLibraryID     int
		TotalBookCount      int
		UncategorizedCount  int
		IsUncategorizedPage bool
	}{
		Books:               books,
		MissingBooks:        nil,
		Collections:         collections,
		Libraries:           libraries,
		ActiveCollectionID:  0,
		ActiveLibraryID:     store.CentralLibraryID,
		TotalBookCount:      totalCount,
		UncategorizedCount:  uncategorizedCount,
		IsUncategorizedPage: true,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

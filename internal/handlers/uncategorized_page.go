package handlers

import (
	"html/template"
	"net/http"

	"folio/internal/store"
)

// UncategorizedPageHandler serves GET /books/uncategorized — books that do not
// belong to any collection.
type UncategorizedPageHandler struct {
	Store    *store.Store
	Template *template.Template
}

func (h *UncategorizedPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	collections, err := h.Store.ListCollections()
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

	// Fetch all thumbnail states in one query to avoid N+1.
	thumbnailSet, err := h.Store.ListBookIDsWithThumbnails()
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
		Collections         []store.Collection
		ActiveCollectionID  int
		TotalBookCount      int
		UncategorizedCount  int
		IsUncategorizedPage bool
	}{
		Books:               books,
		MissingBooks:        nil,
		Collections:         collections,
		ActiveCollectionID:  0,
		TotalBookCount:      totalCount,
		UncategorizedCount:  uncategorizedCount,
		IsUncategorizedPage: true,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

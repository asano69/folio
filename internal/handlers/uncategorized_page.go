package handlers

import (
	"html/template"
	"net/http"

	"folio/internal/storage"
	"folio/internal/store"
)

// UncategorizedPageHandler serves GET /collections/uncategorized.
type UncategorizedPageHandler struct {
	Store     *store.Store
	CachePath string
	Template  *template.Template
}

func (h *UncategorizedPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	collections, err := h.Store.ListBookCollections()
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

	data := shelfPageData{
		PageTitle:          "Uncategorized",
		Books:              books,
		EmptyMessage:       "No uncategorized books.",
		Collections:        collections,
		ActiveCollectionID: "uncategorized",
		TotalBookCount:     totalCount,
		UncategorizedCount: uncategorizedCount,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

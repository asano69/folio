package handlers

import (
	"html/template"
	"net/http"

	"folio/internal/store"
)

type BooksHandler struct {
	Store    *store.Store
	Template *template.Template
}

// bookView is the template model for a single book card.
type bookView struct {
	ID            string
	Title         string
	CoverFilename string
}

func (h *BooksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	books, err := h.Store.ListBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	views := make([]bookView, 0, len(books))
	for _, b := range books {
		bv := bookView{ID: b.ID, Title: b.Title}
		if cover, err := h.Store.GetCoverPage(b.ID); err == nil && cover != nil {
			bv.CoverFilename = cover.Filename
		}
		views = append(views, bv)
	}

	data := struct {
		Books []bookView
	}{
		Books: views,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

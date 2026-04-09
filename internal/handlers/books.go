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
	ID           string
	Title        string
	HasThumbnail bool
	MissingSince string // empty means present; non-empty is the missing-since timestamp
}

func (h *BooksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	books, err := h.Store.ListBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var present, missing []bookView
	for _, b := range books {
		has, _ := h.Store.HasThumbnail(b.ID)
		view := bookView{
			ID:           b.ID,
			Title:        b.Title,
			HasThumbnail: has,
		}
		if b.MissingSince != nil {
			view.MissingSince = *b.MissingSince
			missing = append(missing, view)
		} else {
			present = append(present, view)
		}
	}

	data := struct {
		Books        []bookView
		MissingBooks []bookView
	}{
		Books:        present,
		MissingBooks: missing,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

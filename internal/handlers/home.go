package handlers

import (
	"html/template"
	"net/http"

	"folio/internal/store"
)

// bookView is the template model for a single book card.
// Shared by HomeHandler and CollectionHandler.
type bookView struct {
	ID           string
	Title        string
	HasThumbnail bool
	MissingSince string // empty means present; non-empty is the missing-since timestamp
}

// HomeHandler serves GET / — the all-books library page.
type HomeHandler struct {
	Store    *store.Store
	Template *template.Template
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Reject any path other than "/" to avoid swallowing 404s under this handler.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	collections, err := h.Store.ListCollections()
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

	var present, missing []bookView
	for _, b := range dbBooks {
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
		Books              []bookView
		MissingBooks       []bookView
		Collections        []store.Collection
		ActiveCollectionID int
		TotalBookCount     int
	}{
		Books:              present,
		MissingBooks:       missing,
		Collections:        collections,
		ActiveCollectionID: 0,
		TotalBookCount:     totalCount,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

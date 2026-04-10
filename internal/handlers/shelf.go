package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"folio/internal/store"
)

type ShelfHandler struct {
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

func (h *ShelfHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse optional collection filter.
	collectionID := 0
	if s := r.URL.Query().Get("collection"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			collectionID = n
		}
	}

	collections, err := h.Store.ListCollections()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch books, filtered by collection when one is active.
	var dbBooks []store.Book
	if collectionID > 0 {
		dbBooks, err = h.Store.ListBooksInCollection(collectionID)
	} else {
		dbBooks, err = h.Store.ListBooks()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Total non-missing book count for the "All Books" sidebar label.
	totalCount, err := h.Store.CountAllBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Resolve the active collection's title for the page heading.
	activeTitle := ""
	for _, c := range collections {
		if c.ID == collectionID {
			activeTitle = c.Title
			break
		}
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
		Books                 []bookView
		MissingBooks          []bookView
		Collections           []store.Collection
		ActiveCollectionID    int
		ActiveCollectionTitle string
		TotalBookCount        int
	}{
		Books:                 present,
		MissingBooks:          missing,
		Collections:           collections,
		ActiveCollectionID:    collectionID,
		ActiveCollectionTitle: activeTitle,
		TotalBookCount:        totalCount,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

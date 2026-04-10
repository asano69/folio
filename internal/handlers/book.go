package handlers

import (
	"html/template"
	"net/http"

	"folio/internal/store"
)

// BookHandler renders GET /book?book={id} — a thumbnail grid of all pages.
type BookHandler struct {
	Store    *store.Store
	Template *template.Template
}

// pageGridItem is the template model for a single page card.
type pageGridItem struct {
	Number    int
	Hash      string
	HasThumb  bool
	NoteTitle string
	Attribute string
}

func (h *BookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bookID := r.URL.Query().Get("book")
	if bookID == "" {
		http.Error(w, "book ID required", http.StatusBadRequest)
		return
	}

	book, err := h.Store.GetBook(bookID)
	if err != nil || book == nil {
		http.NotFound(w, r)
		return
	}

	pages, err := h.Store.ListPages(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	notes, err := h.Store.ListNotesByBook(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	thumbSet, err := h.Store.ListPageHashesWithThumbnails(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]pageGridItem, 0, len(pages))
	for _, p := range pages {
		item := pageGridItem{
			Number:   p.Number,
			Hash:     p.Hash,
			HasThumb: thumbSet[p.Hash],
		}
		if n, ok := notes[p.Hash]; ok {
			item.NoteTitle = n.Title
			item.Attribute = n.Attribute
		}
		items = append(items, item)
	}

	data := struct {
		Book  *store.Book
		Pages []pageGridItem
	}{
		Book:  book,
		Pages: items,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

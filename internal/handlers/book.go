package handlers

import (
	"html/template"
	"net/http"

	"folio/internal/store"
)

// BookImagesHandler renders GET /book?book={id} — a thumbnail grid of all images.
type BookImagesHandler struct {
	Store    *store.Store
	Template *template.Template
}

// imageGridItem is the template model for a single image card.
type imageGridItem struct {
	Number    int
	Hash      string
	HasThumb  bool
	NoteTitle string
	Attribute string
}

func (h *BookImagesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	images, err := h.Store.ListImages(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	notes, err := h.Store.ListNotesByBook(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	thumbSet, err := h.Store.ListImageHashesWithThumbnails(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]imageGridItem, 0, len(images))
	for _, img := range images {
		item := imageGridItem{
			Number:   img.Number,
			Hash:     img.Hash,
			HasThumb: thumbSet[img.Hash],
		}
		if n, ok := notes[img.Hash]; ok {
			item.NoteTitle = n.Title
			item.Attribute = n.Attribute
		}
		items = append(items, item)
	}

	data := struct {
		Book   *store.Book
		Images []imageGridItem
	}{
		Book:   book,
		Images: items,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

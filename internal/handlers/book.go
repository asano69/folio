package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"folio/internal/store"
)

// BookDispatchHandler routes /book/{uuid}/overview and /book/{uuid}/bibliographic.
type BookDispatchHandler struct {
	Store                 *store.Store
	OverviewTemplate      *template.Template
	BibliographicTemplate *template.Template
}

// overviewItem is the template model for a single page card in the overview grid.
type overviewItem struct {
	Number    int
	Hash      string
	HasThumb  bool
	NoteTitle string
	Attribute string
	Status    string // always one of: unread, reading, read, skip
}

func (h *BookDispatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Expect /book/{uuid}/overview or /book/{uuid}/bibliographic
	path := strings.TrimPrefix(r.URL.Path, "/book/")
	path = strings.Trim(path, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	bookID, view := parts[0], parts[1]
	switch view {
	case "overview":
		h.serveOverview(w, r, bookID)
	case "bibliographic":
		h.serveBibliographic(w, r, bookID)
	default:
		http.NotFound(w, r)
	}
}

func (h *BookDispatchHandler) serveOverview(w http.ResponseWriter, r *http.Request, bookID string) {
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

	statuses, err := h.Store.ListPageStatuses(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]overviewItem, 0, len(images))
	for _, img := range images {
		status := statuses[img.Hash]
		if status == "" {
			status = "unread"
		}
		item := overviewItem{
			Number:   img.Number,
			Hash:     img.Hash,
			HasThumb: thumbSet[img.Hash],
			Status:   status,
		}
		if n, ok := notes[img.Hash]; ok {
			item.NoteTitle = n.Title
			item.Attribute = n.Attribute
		}
		items = append(items, item)
	}

	data := struct {
		Book  *store.Book
		Items []overviewItem
	}{
		Book:  book,
		Items: items,
	}

	if err := h.OverviewTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *BookDispatchHandler) serveBibliographic(w http.ResponseWriter, r *http.Request, bookID string) {
	book, err := h.Store.GetBook(bookID)
	if err != nil || book == nil {
		http.NotFound(w, r)
		return
	}

	pageCount, err := h.Store.CountPages(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	noteBody, err := h.Store.GetBookNote(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	toc, err := h.Store.GetTOC(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Book      *store.Book
		PageCount int
		NoteBody  string
		TOC       []store.TocEntry
	}{
		Book:      book,
		PageCount: pageCount,
		NoteBody:  noteBody,
		TOC:       toc,
	}

	if err := h.BibliographicTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

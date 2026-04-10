package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"folio/internal/store"
)

// BookDispatchHandler routes /books/{uuid}/overview, /books/{uuid}/bibliography,
// and /books/{uuid}/pages/{page_num}.
type BookDispatchHandler struct {
	Store                 *store.Store
	OverviewTemplate      *template.Template
	BibliographicTemplate *template.Template
	ViewerTemplate        *template.Template
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
	// Expect /books/{uuid}/overview, /books/{uuid}/bibliography,
	// or /books/{uuid}/pages/{num}.
	path := strings.TrimPrefix(r.URL.Path, "/books/")
	path = strings.Trim(path, "/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	bookID, view := parts[0], parts[1]
	switch view {
	case "overview":
		h.serveOverview(w, r, bookID)
	case "bibliography":
		h.serveBibliographic(w, r, bookID)
	case "pages":
		pageNum := 1
		if len(parts) == 3 && parts[2] != "" {
			if n, err := strconv.Atoi(parts[2]); err == nil && n > 0 {
				pageNum = n
			}
		}
		h.serveViewer(w, r, bookID, pageNum)
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

func (h *BookDispatchHandler) serveViewer(w http.ResponseWriter, r *http.Request, bookID string, pageNum int) {
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

	totalPages := len(images)

	if pageNum > totalPages && totalPages > 0 {
		pageNum = totalPages
	}

	var currentImage *store.Image
	if totalPages > 0 {
		currentImage = &images[pageNum-1]
	}

	// Fetch note keyed by the image's content hash, which is stable across
	// re-scans and CBZ image deletions.
	var note store.Note
	if currentImage != nil {
		note, err = h.Store.GetNote(bookID, currentImage.Hash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	toc, err := h.Store.GetTOC(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// ActiveTocIdx is the index of the last section whose page number is <= current page.
	// -1 means no section is active (current page precedes all sections).
	activeTocIdx := -1
	for i, e := range toc {
		if e.PageNum <= pageNum {
			activeTocIdx = i
		}
	}

	data := struct {
		Book         *store.Book
		CurrentImage *store.Image
		Images       []store.Image
		PageNum      int
		TotalPages   int
		HasPrev      bool
		HasNext      bool
		Note         store.Note
		Attributes   []store.AttributeOption
		TOC          []store.TocEntry
		ActiveTocIdx int
	}{
		Book:         book,
		CurrentImage: currentImage,
		Images:       images,
		PageNum:      pageNum,
		TotalPages:   totalPages,
		HasPrev:      pageNum > 1,
		HasNext:      pageNum < totalPages,
		Note:         note,
		Attributes:   store.AllAttributeOptions,
		TOC:          toc,
		ActiveTocIdx: activeTocIdx,
	}

	if err := h.ViewerTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

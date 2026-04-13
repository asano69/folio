package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"folio/internal/storage"
	"folio/internal/store"
)

// BookDispatchHandler routes /books/{uuid}/overview, /books/{uuid}/bibliography,
// and /books/{uuid}/pages/{page_num}.
type BookDispatchHandler struct {
	Store                 *store.Store
	CachePath             string
	OverviewTemplate      *template.Template
	BibliographicTemplate *template.Template
	ViewerTemplate        *template.Template
}

// overviewItem is the template model for a single page card in the overview grid.
type overviewItem struct {
	ID        int // stable page ID, used for thumbnail URL
	Number    int
	HasThumb  bool
	NoteTitle string
	IsSection bool
	Status    string // always one of: unread, reading, read, skip
}

func (h *BookDispatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	pages, err := h.Store.ListPages(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Page thumbnail existence is keyed by stable page ID.
	thumbSet, err := storage.ListPageThumbnailIDs(h.CachePath, bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Page statuses are keyed by stable page ID.
	statuses, err := h.Store.ListPageStatuses(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Note titles and section markers are fetched in bulk to avoid per-page queries.
	noteTitles, err := h.Store.ListPageNoteTitlesByBook(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sectionPageIDs, err := h.Store.ListPageSectionPageIDsByBook(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	items := make([]overviewItem, 0, len(pages))
	for _, p := range pages {
		status := statuses[p.ID]
		if status == "" {
			status = "unread"
		}
		items = append(items, overviewItem{
			ID:        p.ID,
			Number:    p.Number,
			HasThumb:  thumbSet[p.ID],
			NoteTitle: noteTitles[p.ID],
			IsSection: sectionPageIDs[p.ID],
			Status:    status,
		})
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

	pages, err := h.Store.ListPages(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalPages := len(pages)

	if pageNum > totalPages && totalPages > 0 {
		pageNum = totalPages
	}

	var currentPage *store.Page
	if totalPages > 0 {
		currentPage = &pages[pageNum-1]
	}

	// Fetch note, drawing, and section info for the current page.
	var noteTitle, noteBody, svgDrawing, sectionTitle string
	var isSection bool
	if currentPage != nil {
		note, err := h.Store.GetPageNote(currentPage.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		noteTitle = note.Title
		noteBody = note.Body

		svgDrawing, err = h.Store.GetPageDrawing(currentPage.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		section, err := h.Store.GetPageSection(currentPage.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if section != nil {
			isSection = true
			sectionTitle = section.Title
		}
	}

	toc, err := h.Store.GetTOC(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// ActiveTocIdx is the index of the last section whose page number is <=
	// the current page. -1 means no section is active.
	activeTocIdx := -1
	for i, e := range toc {
		if e.PageNum <= pageNum {
			activeTocIdx = i
		}
	}

	data := struct {
		Book         *store.Book
		CurrentPage  *store.Page
		Pages        []store.Page
		PageNum      int
		TotalPages   int
		HasPrev      bool
		HasNext      bool
		NoteTitle    string
		NoteBody     string
		SvgDrawing   string
		IsSection    bool
		SectionTitle string
		TOC          []store.TocEntry
		ActiveTocIdx int
	}{
		Book:         book,
		CurrentPage:  currentPage,
		Pages:        pages,
		PageNum:      pageNum,
		TotalPages:   totalPages,
		HasPrev:      pageNum > 1,
		HasNext:      pageNum < totalPages,
		NoteTitle:    noteTitle,
		NoteBody:     noteBody,
		SvgDrawing:   svgDrawing,
		IsSection:    isSection,
		SectionTitle: sectionTitle,
		TOC:          toc,
		ActiveTocIdx: activeTocIdx,
	}

	if err := h.ViewerTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"folio/internal/storage"
	"folio/internal/store"
)

// BookDispatchHandler routes:
//
//	GET /books/{uuid}/overview      — page grid with status and thumbnails
//	GET /books/{uuid}/bibliography  — TOC, stats, and book-level memo
//	GET /books/{uuid}?seq=N         — viewer at image sequence position N
//	GET /books/{uuid}?p=LABEL       — viewer at the image carrying book page label LABEL
//	GET /books/{uuid}               — redirect to /books/{uuid}/overview
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
	Seq       int
	HasThumb  bool
	IsSection bool
	Status    string // always one of: unread, reading, read, skip
}

func (h *BookDispatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/books/")
	path = strings.Trim(path, "/")
	parts := strings.SplitN(path, "/", 2)
	bookID := parts[0]

	if len(parts) == 1 {
		// /books/{uuid} — dispatch on query parameters.
		if seqStr := r.URL.Query().Get("seq"); seqStr != "" {
			seq, err := strconv.Atoi(seqStr)
			if err != nil || seq < 1 {
				http.NotFound(w, r)
				return
			}
			h.serveViewer(w, r, bookID, seq)
			return
		}
		if label := r.URL.Query().Get("p"); label != "" {
			h.serveViewerByLabel(w, r, bookID, label)
			return
		}
		// No recognised query parameter: redirect to the overview page.
		http.Redirect(w, r, "/books/"+bookID+"/overview", http.StatusFound)
		return
	}

	switch parts[1] {
	case "overview":
		h.serveOverview(w, r, bookID)
	case "bibliography":
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
			Seq:       p.Seq,
			HasThumb:  thumbSet[p.ID],
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

// serveViewerByLabel looks up the page carrying the given book-page label and
// delegates to serveViewer. Returns 404 if no page has that label.
func (h *BookDispatchHandler) serveViewerByLabel(w http.ResponseWriter, r *http.Request, bookID, label string) {
	page, err := h.Store.GetPageByLabel(bookID, label)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if page == nil {
		http.NotFound(w, r)
		return
	}
	h.serveViewer(w, r, bookID, page.Seq)
}

func (h *BookDispatchHandler) serveViewer(w http.ResponseWriter, r *http.Request, bookID string, seq int) {
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

	// Find the page matching the requested seq.
	currentIdx := -1
	for i, p := range pages {
		if p.Seq == seq {
			currentIdx = i
			break
		}
	}
	// If seq not found, redirect to the first page.
	if currentIdx == -1 {
		if len(pages) > 0 {
			http.Redirect(w, r, fmt.Sprintf("/books/%s?seq=%d", bookID, pages[0].Seq), http.StatusFound)
		} else {
			http.NotFound(w, r)
		}
		return
	}

	currentPage := &pages[currentIdx]

	hasPrev := currentIdx > 0
	hasNext := currentIdx < len(pages)-1
	var prevSeq, nextSeq int
	if hasPrev {
		prevSeq = pages[currentIdx-1].Seq
	}
	if hasNext {
		nextSeq = pages[currentIdx+1].Seq
	}

	// Fetch note, drawing, and section info for the current page.
	var noteBody, svgDrawing, sectionTitle, sectionDescription string
	var isSection bool

	note, err := h.Store.GetPageNote(currentPage.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
		sectionDescription = section.Description
	}

	toc, err := h.Store.GetTOC(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// ActiveTocIdx is the index of the last section whose seq is <= current seq.
	// -1 means no section is active.
	activeTocIdx := -1
	for i, e := range toc {
		if e.PageNum <= seq {
			activeTocIdx = i
		}
	}

	data := struct {
		Book               *store.Book
		CurrentPage        *store.Page
		Pages              []store.Page
		Seq                int
		TotalPages         int
		HasPrev            bool
		HasNext            bool
		PrevSeq            int
		NextSeq            int
		NoteBody           string
		SvgDrawing         string
		IsSection          bool
		SectionTitle       string
		SectionDescription string
		TOC                []store.TocEntry
		ActiveTocIdx       int
	}{
		Book:               book,
		CurrentPage:        currentPage,
		Pages:              pages,
		Seq:                seq,
		TotalPages:         len(pages),
		HasPrev:            hasPrev,
		HasNext:            hasNext,
		PrevSeq:            prevSeq,
		NextSeq:            nextSeq,
		NoteBody:           noteBody,
		SvgDrawing:         svgDrawing,
		IsSection:          isSection,
		SectionTitle:       sectionTitle,
		SectionDescription: sectionDescription,
		TOC:                toc,
		ActiveTocIdx:       activeTocIdx,
	}

	if err := h.ViewerTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

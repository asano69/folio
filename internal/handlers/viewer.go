package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"folio/internal/store"
)

type ViewerHandler struct {
	Store    *store.Store
	Template *template.Template
}

func (h *ViewerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bookID := r.URL.Query().Get("book")
	pageNumStr := r.URL.Query().Get("page")

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

	totalPages := len(pages)

	pageNum := 1
	if pageNumStr != "" {
		if n, err := strconv.Atoi(pageNumStr); err == nil && n > 0 {
			pageNum = n
		}
	}
	if pageNum > totalPages {
		pageNum = totalPages
	}

	var currentPage *store.Page
	if totalPages > 0 {
		currentPage = &pages[pageNum-1]
	}

	// Fetch note keyed by the page's content hash, which is stable across
	// re-scans and CBZ page deletions.
	var note store.Note
	if currentPage != nil {
		note, err = h.Store.GetNote(bookID, currentPage.Hash)
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
		CurrentPage  *store.Page
		Pages        []store.Page
		PageNum      int
		TotalPages   int
		HasPrev      bool
		HasNext      bool
		Note         store.Note
		Attributes   []store.AttributeOption
		TOC          []store.TOCEntry
		ActiveTocIdx int
	}{
		Book:         book,
		CurrentPage:  currentPage,
		Pages:        pages,
		PageNum:      pageNum,
		TotalPages:   totalPages,
		HasPrev:      pageNum > 1,
		HasNext:      pageNum < totalPages,
		Note:         note,
		Attributes:   store.AllAttributeOptions,
		TOC:          toc,
		ActiveTocIdx: activeTocIdx,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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

	images, err := h.Store.ListImages(bookID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalPages := len(images)

	pageNum := 1
	if pageNumStr != "" {
		if n, err := strconv.Atoi(pageNumStr); err == nil && n > 0 {
			pageNum = n
		}
	}
	if pageNum > totalPages {
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

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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

	pageNum := 1
	if pageNumStr != "" {
		if n, err := strconv.Atoi(pageNumStr); err == nil && n > 0 {
			pageNum = n
		}
	}

	// Clamp page number to valid range.
	if pageNum < 1 {
		pageNum = 1
	}
	if pageNum > len(pages) {
		pageNum = len(pages)
	}

	var currentPage *store.Page
	if pageNum > 0 && pageNum <= len(pages) {
		currentPage = &pages[pageNum-1]
	}

	data := struct {
		Book        *store.Book
		Pages       []store.Page
		CurrentPage *store.Page
		PageNum     int
		TotalPages  int
		HasPrev     bool
		HasNext     bool
	}{
		Book:        book,
		Pages:       pages,
		CurrentPage: currentPage,
		PageNum:     pageNum,
		TotalPages:  len(pages),
		HasPrev:     pageNum > 1,
		HasNext:     pageNum < len(pages),
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

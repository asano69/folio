package handlers

import (
	"html/template"
	"net/http"
	"folio/internal/library"
	"strconv"
)

type ViewerHandler struct {
	Library  *library.Library
	Template *template.Template
}

func (h *ViewerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bookID := r.URL.Query().Get("book")
	pageNumStr := r.URL.Query().Get("page")

	if bookID == "" {
		http.Error(w, "Book ID required", http.StatusBadRequest)
		return
	}

	book := h.Library.GetBook(bookID)
	if book == nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	pageNum := 1
	if pageNumStr != "" {
		if num, err := strconv.Atoi(pageNumStr); err == nil && num > 0 {
			pageNum = num
		}
	}

	// ページ番号を範囲内に制限
	if pageNum < 1 {
		pageNum = 1
	}
	if pageNum > len(book.Pages) {
		pageNum = len(book.Pages)
	}

	var currentPage *library.Page
	if pageNum > 0 && pageNum <= len(book.Pages) {
		currentPage = &book.Pages[pageNum-1]
	}

	data := struct {
		Book        *library.Book
		CurrentPage *library.Page
		PageNum     int
		TotalPages  int
		HasPrev     bool
		HasNext     bool
	}{
		Book:        book,
		CurrentPage: currentPage,
		PageNum:     pageNum,
		TotalPages:  len(book.Pages),
		HasPrev:     pageNum > 1,
		HasNext:     pageNum < len(book.Pages),
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

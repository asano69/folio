package handlers

import (
	"html/template"
	"net/http"
	"openbook/internal/library"
)

type BooksHandler struct {
	Library  *library.Library
	Template *template.Template
}

func (h *BooksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Books []library.Book
	}{
		Books: h.Library.Books,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

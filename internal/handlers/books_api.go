package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"folio/internal/storage"
	"folio/internal/store"
)

// BooksAPIHandler handles REST API requests under /api/books/.
//
// Routes:
//
//	PUT  /api/books/{id}           — rename a book (updates folio.json + DB title)
//	PUT  /api/books/{id}/meta      — save all book metadata (updates folio.json + DB)
//	POST /api/books/{id}/thumbnail — regenerate book-level thumbnail
type BooksAPIHandler struct {
	Store     *store.Store
	CachePath string
}

func (h *BooksAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/books/")

	// POST /api/books/{id}/thumbnail
	if strings.HasSuffix(path, "/thumbnail") {
		bookID := strings.TrimSuffix(path, "/thumbnail")
		if bookID == "" || strings.Contains(bookID, "/") {
			http.Error(w, "invalid book ID", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.regenerateThumbnail(w, r, bookID)
		return
	}

	// PUT /api/books/{id}/meta
	if strings.HasSuffix(path, "/meta") {
		bookID := strings.TrimSuffix(path, "/meta")
		if bookID == "" || strings.Contains(bookID, "/") {
			http.Error(w, "invalid book ID", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.saveBookMeta(w, r, bookID)
		return
	}

	// PUT /api/books/{id}
	bookID := strings.Trim(path, "/")
	if bookID == "" || strings.Contains(bookID, "/") {
		http.Error(w, "invalid book ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.renameBook(w, r, bookID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *BooksAPIHandler) renameBook(w http.ResponseWriter, r *http.Request, bookID string) {
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(body.Title)
	if title == "" {
		http.Error(w, "title cannot be empty", http.StatusBadRequest)
		return
	}

	book, err := h.Store.GetBook(bookID)
	if err != nil || book == nil {
		http.NotFound(w, r)
		return
	}

	// Update folio.json inside the CBZ first; if this fails the DB is not touched.
	if err := storage.UpdateTitle(book.Source, title); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.Store.UpdateBookTitle(bookID, title); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}{ID: bookID, Title: title})
}

// bookMetaRequest is the request body for PUT /api/books/{id}/meta.
// All fields except title are optional.
type bookMetaRequest struct {
	Title        string               `json:"title"`
	Type         string               `json:"type"`
	Abstract     string               `json:"abstract"`
	Language     string               `json:"language"`
	Author       []storage.PersonName `json:"author"`
	Translator   []storage.PersonName `json:"translator"`
	OrigTitle    string               `json:"origtitle"`
	Edition      string               `json:"edition"`
	Volume       string               `json:"volume"`
	Series       string               `json:"series"`
	SeriesNumber string               `json:"series_number"`
	Publisher    string               `json:"publisher"`
	Year         string               `json:"year"`
	Note         string               `json:"note"`
	Keywords     []string             `json:"keywords"`
	ISBN         string               `json:"isbn"`
	Links        []string             `json:"links"`
}

// saveBookMeta handles PUT /api/books/{id}/meta.
// It writes all metadata fields to both folio.json inside the CBZ and the DB.
func (h *BooksAPIHandler) saveBookMeta(w http.ResponseWriter, r *http.Request, bookID string) {
	var req bookMetaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		http.Error(w, "title cannot be empty", http.StatusBadRequest)
		return
	}

	book, err := h.Store.GetBook(bookID)
	if err != nil || book == nil {
		http.NotFound(w, r)
		return
	}

	meta := storage.Book{
		ID:           bookID,
		Title:        title,
		Type:         req.Type,
		Abstract:     req.Abstract,
		Language:     req.Language,
		Author:       req.Author,
		Translator:   req.Translator,
		OrigTitle:    req.OrigTitle,
		Edition:      req.Edition,
		Volume:       req.Volume,
		Series:       req.Series,
		SeriesNumber: req.SeriesNumber,
		Publisher:    req.Publisher,
		Year:         req.Year,
		Note:         req.Note,
		Keywords:     req.Keywords,
		ISBN:         req.ISBN,
		Links:        req.Links,
	}

	// Update folio.json inside the CBZ first; if this fails the DB is not touched.
	if err := storage.UpdateFolioMeta(book.Source, meta); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.Store.UpdateBookMeta(bookID, meta); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// regenerateThumbnail handles POST /api/books/{id}/thumbnail.
func (h *BooksAPIHandler) regenerateThumbnail(w http.ResponseWriter, r *http.Request, bookID string) {
	book, err := h.Store.GetBook(bookID)
	if err != nil || book == nil {
		http.NotFound(w, r)
		return
	}

	data, err := storage.GenerateBookThumbnail(book.Source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := storage.WriteBookThumbnail(h.CachePath, bookID, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

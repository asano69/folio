package handlers

import (
	"encoding/json"
	"folio/internal/store"
	"net/http"
	"regexp"
	"strings"
)

// PagesAPIHandler handles:
//
//	PUT /api/pages/{bookID}/{pageHash}          — save text note
//	PUT /api/pages/{bookID}/{pageHash}/drawing  — save SVG drawing
//	PUT /api/pages/{bookID}/{pageHash}/status   — update read status
type PagesAPIHandler struct {
	Store *store.Store
}

// verifyPageHashExists checks that the given pageHash is registered for the book.
// Returns the page number if valid, or -1 if not found.
func (h *PagesAPIHandler) verifyPageHashExists(bookID, pageHash string) (int, error) {
	img, err := h.Store.GetImageByHash(bookID, pageHash)
	if err != nil {
		return -1, err
	}
	if img == nil {
		return -1, nil
	}
	return img.Number, nil
}

// isSVGWellFormed performs a basic check that the SVG markup is not obviously malformed.
// This is a simple heuristic check; full SVG validation is expensive and handled by the browser.
func isSVGWellFormed(svg string) bool {
	if svg == "" {
		return true // Empty string is valid (clears drawing).
	}

	// Count opening and closing <g> and <path> tags.
	openG := regexp.MustCompile(`<g[^>]*>`).FindAllString(svg, -1)
	closeG := regexp.MustCompile(`</g>`).FindAllString(svg, -1)

	if len(openG) != len(closeG) {
		return false
	}

	// Check for obvious XSS patterns (defensive, since DB is internal use only).
	if regexp.MustCompile(`(javascript|onerror|onload|onclick)`).MatchString(svg) {
		return false
	}

	return true
}

func (h *PagesAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/pages/")

	// PUT /api/pages/{bookID}/{pageHash}/drawing
	if strings.HasSuffix(path, "/drawing") {
		inner := strings.TrimSuffix(path, "/drawing")
		parts := strings.SplitN(inner, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.saveDrawing(w, r, parts[0], parts[1])
		return
	}

	// PUT /api/pages/{bookID}/{pageHash}/status
	if strings.HasSuffix(path, "/status") {
		inner := strings.TrimSuffix(path, "/status")
		parts := strings.SplitN(inner, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.saveStatus(w, r, parts[0], parts[1])
		return
	}

	// PUT /api/pages/{bookID}/{pageHash}
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	bookID := parts[0]
	pageHash := parts[1]

	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Title     string `json:"title"`
		Attribute string `json:"attribute"`
		Body      string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	book, err := h.Store.GetBook(bookID)
	if err != nil || book == nil {
		http.NotFound(w, r)
		return
	}

	// Verify that the pageHash is valid for this book.
	pageNum, err := h.verifyPageHashExists(bookID, pageHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if pageNum == -1 {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}

	note := store.Note{
		BookID:    bookID,
		PageHash:  pageHash,
		Title:     strings.TrimSpace(body.Title),
		Attribute: body.Attribute,
		Body:      body.Body,
	}

	if err := h.Store.UpsertNote(note); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(note)
}

func (h *PagesAPIHandler) saveDrawing(w http.ResponseWriter, r *http.Request, bookID, pageHash string) {
	var body struct {
		SvgDrawing *string `json:"svg_drawing"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	book, err := h.Store.GetBook(bookID)
	if err != nil || book == nil {
		http.NotFound(w, r)
		return
	}

	pageNum, err := h.verifyPageHashExists(bookID, pageHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if pageNum == -1 {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}

	// Validate SVG if present.
	if body.SvgDrawing != nil && *body.SvgDrawing != "" {
		if !isSVGWellFormed(*body.SvgDrawing) {
			http.Error(w, "SVG markup is malformed", http.StatusBadRequest)
			return
		}
	}

	if err := h.Store.UpsertDrawing(bookID, pageHash, body.SvgDrawing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

var validStatuses = map[string]bool{
	"unread": true, "reading": true, "read": true, "skip": true,
}

func (h *PagesAPIHandler) saveStatus(w http.ResponseWriter, r *http.Request, bookID, pageHash string) {
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !validStatuses[body.Status] {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	book, err := h.Store.GetBook(bookID)
	if err != nil || book == nil {
		http.NotFound(w, r)
		return
	}

	// Verify that the pageHash is valid for this book.
	pageNum, err := h.verifyPageHashExists(bookID, pageHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if pageNum == -1 {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}

	if err := h.Store.UpsertPageStatus(bookID, pageHash, body.Status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"folio/internal/store"
)

// NoteAPIHandler handles:
//
//	PUT /api/pages/{bookID}/{pageHash}          — save text note
//	PUT /api/pages/{bookID}/{pageHash}/drawing  — save SVG drawing
type NoteAPIHandler struct {
	Store *store.Store
}

func (h *NoteAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	// svg_drawing is intentionally absent from this endpoint.
	// Drawings are saved separately via PUT /api/pages/{bookID}/{pageHash}/drawing
	// to prevent a text note save from accidentally clearing an existing drawing.
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

func (h *NoteAPIHandler) saveDrawing(w http.ResponseWriter, r *http.Request, bookID, pageHash string) {
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

	if err := h.Store.UpsertDrawing(bookID, pageHash, body.SvgDrawing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

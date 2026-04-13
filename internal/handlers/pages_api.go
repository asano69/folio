package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"folio/internal/store"
)

// PagesAPIHandler handles:
//
//	PUT /api/pages/{pageID}          — save title, attribute, and note body
//	PUT /api/pages/{pageID}/drawing  — save or clear SVG drawing
//	PUT /api/pages/{pageID}/status   — update read status
//
// The page ID is the stable integer primary key from the pages table.
// It remains valid across re-scans and CBZ modifications.
type PagesAPIHandler struct {
	Store *store.Store
}

func (h *PagesAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/pages/")

	// PUT /api/pages/{pageID}/drawing
	if strings.HasSuffix(path, "/drawing") {
		pageID, err := parsePageID(strings.TrimSuffix(path, "/drawing"))
		if err != nil {
			http.Error(w, "invalid page ID", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.saveDrawing(w, r, pageID)
		return
	}

	// PUT /api/pages/{pageID}/status
	if strings.HasSuffix(path, "/status") {
		pageID, err := parsePageID(strings.TrimSuffix(path, "/status"))
		if err != nil {
			http.Error(w, "invalid page ID", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.saveStatus(w, r, pageID)
		return
	}

	// PUT /api/pages/{pageID}
	pageID, err := parsePageID(path)
	if err != nil {
		http.Error(w, "invalid page ID", http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.savePageEdit(w, r, pageID)
}

// parsePageID strips surrounding slashes and converts the segment to int.
func parsePageID(s string) (int, error) {
	return strconv.Atoi(strings.Trim(s, "/"))
}

// savePageEdit handles PUT /api/pages/{pageID}.
// Updates title and attribute on the pages row and upserts the note body.
func (h *PagesAPIHandler) savePageEdit(w http.ResponseWriter, r *http.Request, pageID int) {
	var body struct {
		Title     string `json:"title"`
		Attribute string `json:"attribute"`
		Body      string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	page, err := h.Store.GetPage(pageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if page == nil {
		http.NotFound(w, r)
		return
	}

	if err := h.Store.UpsertPageEdit(pageID, strings.TrimSpace(body.Title), body.Attribute, body.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// saveDrawing handles PUT /api/pages/{pageID}/drawing.
func (h *PagesAPIHandler) saveDrawing(w http.ResponseWriter, r *http.Request, pageID int) {
	var body struct {
		SvgDrawing *string `json:"svg_drawing"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	page, err := h.Store.GetPage(pageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if page == nil {
		http.NotFound(w, r)
		return
	}

	if body.SvgDrawing != nil && *body.SvgDrawing != "" {
		if !isSVGWellFormed(*body.SvgDrawing) {
			http.Error(w, "SVG markup is malformed", http.StatusBadRequest)
			return
		}
	}

	if err := h.Store.UpsertPageDrawing(pageID, body.SvgDrawing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

var validStatuses = map[string]bool{
	"unread": true, "reading": true, "read": true, "skip": true,
}

// saveStatus handles PUT /api/pages/{pageID}/status.
func (h *PagesAPIHandler) saveStatus(w http.ResponseWriter, r *http.Request, pageID int) {
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

	page, err := h.Store.GetPage(pageID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if page == nil {
		http.NotFound(w, r)
		return
	}

	if err := h.Store.UpsertPageStatus(pageID, body.Status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// isSVGWellFormed performs a basic sanity check on SVG markup.
// Full validation is handled by the browser; this is a lightweight guard.
func isSVGWellFormed(svg string) bool {
	if svg == "" {
		return true
	}
	openG := regexp.MustCompile(`<g[^>]*>`).FindAllString(svg, -1)
	closeG := regexp.MustCompile(`</g>`).FindAllString(svg, -1)
	if len(openG) != len(closeG) {
		return false
	}
	if regexp.MustCompile(`(javascript|onerror|onload|onclick)`).MatchString(svg) {
		return false
	}
	return true
}

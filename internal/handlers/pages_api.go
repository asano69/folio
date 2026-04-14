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
//	PUT /api/pages/{pageID}/note        — save note body
//	PUT /api/pages/{pageID}/drawing     — save or clear SVG drawing
//	PUT /api/pages/{pageID}/status      — update read status
//	PUT /api/pages/{pageID}/page-number — set or clear the real book page number
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

	// PUT /api/pages/{pageID}/note
	if strings.HasSuffix(path, "/note") {
		pageID, err := parsePageID(strings.TrimSuffix(path, "/note"))
		if err != nil {
			http.Error(w, "invalid page ID", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.savePageNote(w, r, pageID)
		return
	}

	// PUT /api/pages/{pageID}/page-number
	if strings.HasSuffix(path, "/page-number") {
		pageID, err := parsePageID(strings.TrimSuffix(path, "/page-number"))
		if err != nil {
			http.Error(w, "invalid page ID", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.savePageNumber(w, r, pageID)
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

// parsePageID strips surrounding slashes and converts the segment to int.
func parsePageID(s string) (int, error) {
	return strconv.Atoi(strings.Trim(s, "/"))
}

// savePageNote handles PUT /api/pages/{pageID}/note.
func (h *PagesAPIHandler) savePageNote(w http.ResponseWriter, r *http.Request, pageID int) {
	var body struct {
		Body string `json:"body"`
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

	if err := h.Store.UpsertPageNote(pageID, body.Body); err != nil {
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

// savePageNumber handles PUT /api/pages/{pageID}/page-number.
// The page_number field stores the real book page number as printed (e.g. "42",
// "iv"). Pass null in the JSON body to clear an existing value.
func (h *PagesAPIHandler) savePageNumber(w http.ResponseWriter, r *http.Request, pageID int) {
	var body struct {
		PageNumber *string `json:"page_number"`
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

	if err := h.Store.UpdatePageNumber(pageID, body.PageNumber); err != nil {
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

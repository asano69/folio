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
//	PUT /api/pages/{pageID}/note     — save note body
//	PUT /api/pages/{pageID}/section  — mark or unmark a page as a section start
//	PUT /api/pages/{pageID}/drawing  — save or clear SVG drawing
//	PUT /api/pages/{pageID}/status   — update read status
//	PUT /api/pages/{pageID}/labels   — replace all book-page-number labels
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

	// PUT /api/pages/{pageID}/section
	if strings.HasSuffix(path, "/section") {
		pageID, err := parsePageID(strings.TrimSuffix(path, "/section"))
		if err != nil {
			http.Error(w, "invalid page ID", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.savePageSection(w, r, pageID)
		return
	}

	// PUT /api/pages/{pageID}/labels
	if strings.HasSuffix(path, "/labels") {
		pageID, err := parsePageID(strings.TrimSuffix(path, "/labels"))
		if err != nil {
			http.Error(w, "invalid page ID", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.saveLabels(w, r, pageID)
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

// parsePageID strips surrounding slashes and converts the segment to int.
func parsePageID(s string) (int, error) {
	return strconv.Atoi(strings.Trim(s, "/"))
}

// savePageNote handles PUT /api/pages/{pageID}/note.
// Updates the note body for a page.
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

// savePageSection handles PUT /api/pages/{pageID}/section.
// When enabled is true, the page is marked as a section start with the given
// title and description. When enabled is false, the section marking is removed.
func (h *PagesAPIHandler) savePageSection(w http.ResponseWriter, r *http.Request, pageID int) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
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

	if body.Enabled {
		if err := h.Store.UpsertPageSection(pageID, strings.TrimSpace(body.Title), body.Description); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.Store.DeletePageSection(pageID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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

// saveLabels handles PUT /api/pages/{pageID}/labels.
// Replaces all book-page-number labels for a page. An empty array removes all labels.
func (h *PagesAPIHandler) saveLabels(w http.ResponseWriter, r *http.Request, pageID int) {
	var body struct {
		Labels []string `json:"labels"`
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

	if err := h.Store.UpsertPageLabels(pageID, body.Labels); err != nil {
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

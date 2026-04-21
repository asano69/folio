package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"folio/internal/store"
)

// LibraryPageHandler serves GET /library — the library management page.
// It shows a two-column layout: a library list on the left and collection
// tiles for the selected library on the right.
//
// The active library is selected via the ?lib= query parameter.
// Defaults to Central Library when the parameter is absent or invalid.
type LibraryPageHandler struct {
	Store    *store.Store
	Template *template.Template
}

func (h *LibraryPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	libraries, err := h.Store.ListLibraries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Determine the active library from the query parameter.
	activeLibraryID := store.CentralLibraryID
	if libStr := r.URL.Query().Get("lib"); libStr != "" {
		if id, err := strconv.Atoi(libStr); err == nil && id > 0 {
			activeLibraryID = id
		}
	}

	// Verify the requested library exists; fall back to Central Library if not.
	found := false
	for _, lib := range libraries {
		if lib.ID == activeLibraryID {
			found = true
			break
		}
	}
	if !found {
		activeLibraryID = store.CentralLibraryID
	}

	collections, err := h.Store.ListBookCollectionsInLibrary(activeLibraryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Determine the display name for the active library.
	activeLibraryName := "Central Library"
	for _, lib := range libraries {
		if lib.ID == activeLibraryID {
			activeLibraryName = lib.Name
			break
		}
	}

	data := struct {
		Libraries         []store.Library
		ActiveLibraryID   int
		ActiveLibraryName string
		Collections       []store.BookCollection
		// IsCentralLibrary is true when the active library is Central Library.
		// Used to conditionally disable rename/delete controls in the template.
		IsCentralLibrary bool
	}{
		Libraries:         libraries,
		ActiveLibraryID:   activeLibraryID,
		ActiveLibraryName: activeLibraryName,
		Collections:       collections,
		IsCentralLibrary:  activeLibraryID == store.CentralLibraryID,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

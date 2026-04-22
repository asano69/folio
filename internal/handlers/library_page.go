package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"folio/internal/store"
)

// libraryNavItem is the template model for a library sidebar entry.
type libraryNavItem struct {
	ID              string
	Name            string
	CollectionCount int
	IsCentral       bool
	URL             string
}

// collectionTileItem is the template model for a collection tile in the library admin.
type collectionTileItem struct {
	ID        string
	Name      string
	BookCount int
	URL       string
}

// LibraryPageHandler serves GET /libraries/all and GET /libraries/{uuid}.
type LibraryPageHandler struct {
	Store    *store.Store
	Template *template.Template
}

func (h *LibraryPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	seg := strings.TrimPrefix(r.URL.Path, "/libraries/")
	seg = strings.Trim(seg, "/")

	// Determine the active library.
	activeLibraryID := store.CentralLibraryID
	if seg != "" && seg != "all" {
		activeLibraryID = seg
	}

	libraries, err := h.Store.ListLibraries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Verify the requested library exists (only needed for non-Central).
	if activeLibraryID != store.CentralLibraryID {
		found := false
		for _, lib := range libraries {
			if lib.ID == activeLibraryID {
				found = true
				break
			}
		}
		if !found {
			http.NotFound(w, r)
			return
		}
	}

	rawCollections, err := h.Store.ListBookCollectionsInLibrary(activeLibraryID)
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

	// Build view types with precomputed URLs.
	libraryItems := make([]libraryNavItem, 0, len(libraries))
	for _, lib := range libraries {
		isCentral := lib.ID == store.CentralLibraryID
		url := "/libraries/" + lib.ID
		if isCentral {
			url = "/libraries/all"
		}
		libraryItems = append(libraryItems, libraryNavItem{
			ID:              lib.ID,
			Name:            lib.Name,
			CollectionCount: lib.CollectionCount,
			IsCentral:       isCentral,
			URL:             url,
		})
	}

	collectionItems := make([]collectionTileItem, 0, len(rawCollections))
	for _, c := range rawCollections {
		collectionItems = append(collectionItems, collectionTileItem{
			ID:        c.ID,
			Name:      c.Name,
			BookCount: c.BookCount,
			URL:       "/collections/" + c.ID,
		})
	}

	data := struct {
		Libraries         []libraryNavItem
		ActiveLibraryID   string
		ActiveLibraryName string
		Collections       []collectionTileItem
		IsCentralLibrary  bool
	}{
		Libraries:         libraryItems,
		ActiveLibraryID:   activeLibraryID,
		ActiveLibraryName: activeLibraryName,
		Collections:       collectionItems,
		IsCentralLibrary:  activeLibraryID == store.CentralLibraryID,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

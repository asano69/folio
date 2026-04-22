package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"folio/internal/storage"
	"folio/internal/store"
)

// CollectionPageHandler serves GET /collections/{uuid} — a single book collection.
type CollectionPageHandler struct {
	Store     *store.Store
	CachePath string
	Template  *template.Template
}

func (h *CollectionPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	collectionID := strings.TrimPrefix(r.URL.Path, "/collections/")
	collectionID = strings.Trim(collectionID, "/")

	if collectionID == "" {
		http.Redirect(w, r, "/collections/all", http.StatusFound)
		return
	}

	activeCollection, err := h.Store.GetBookCollection(collectionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if activeCollection == nil {
		http.NotFound(w, r)
		return
	}

	libID := r.URL.Query().Get("lib")
	if libID == "" {
		libID = store.CentralLibraryID
	}

	libraries, err := h.Store.ListLibraries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	collections, err := h.Store.ListBookCollectionsInLibrary(libID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dbBooks, err := h.Store.ListBooksInBookCollection(collectionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalCount, err := h.Store.CountAllBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	uncategorizedCount, err := h.Store.CountUncategorizedBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	thumbnailSet, err := storage.ListBookThumbnailIDs(h.CachePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var present, missing []bookView
	for _, b := range dbBooks {
		view := bookView{
			ID:           b.ID,
			Title:        b.Title,
			HasThumbnail: thumbnailSet[b.ID],
		}
		if b.MissingSince != nil {
			view.MissingSince = *b.MissingSince
			missing = append(missing, view)
		} else {
			present = append(present, view)
		}
	}

	data := shelfPageData{
		PageTitle:          activeCollection.Name,
		Books:              present,
		MissingBooks:       missing,
		EmptyMessage:       "No books in this collection yet.",
		Collections:        collections,
		ActiveCollectionID: collectionID,
		CollectionID:       collectionID,
		TotalBookCount:     totalCount,
		UncategorizedCount: uncategorizedCount,
		Libraries:       libraries,
		ActiveLibraryID: libID,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

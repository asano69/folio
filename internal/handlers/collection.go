package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"folio/internal/store"
)

// CollectionHandler serves GET /collections/{id} — a single collection's book list.
type CollectionHandler struct {
	Store    *store.Store
	Template *template.Template
}

func (h *CollectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/collections/")
	idStr = strings.Trim(idStr, "/")

	// /collections/ with no ID redirects to home.
	if idStr == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	collectionID, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	collections, err := h.Store.ListCollections()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Locate the active collection within the already-fetched list to avoid a
	// second DB query. 404 if the ID does not exist.
	var activeCollection *store.Collection
	for i := range collections {
		if collections[i].ID == collectionID {
			activeCollection = &collections[i]
			break
		}
	}
	if activeCollection == nil {
		http.NotFound(w, r)
		return
	}

	dbBooks, err := h.Store.ListBooksInCollection(collectionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	totalCount, err := h.Store.CountAllBooks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var present, missing []bookView
	for _, b := range dbBooks {
		has, _ := h.Store.HasThumbnail(b.ID)
		view := bookView{
			ID:           b.ID,
			Title:        b.Title,
			HasThumbnail: has,
		}
		if b.MissingSince != nil {
			view.MissingSince = *b.MissingSince
			missing = append(missing, view)
		} else {
			present = append(present, view)
		}
	}

	data := struct {
		Books              []bookView
		MissingBooks       []bookView
		Collections        []store.Collection
		ActiveCollectionID int
		Collection         *store.Collection
		TotalBookCount     int
	}{
		Books:              present,
		MissingBooks:       missing,
		Collections:        collections,
		ActiveCollectionID: collectionID,
		Collection:         activeCollection,
		TotalBookCount:     totalCount,
	}

	if err := h.Template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

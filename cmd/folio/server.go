package main

import (
	"fmt"
	"html/template"
	"net/http"

	"folio/internal/config"
	"folio/internal/handlers"
	"folio/internal/store"
)

type server struct {
	config *config.Config
	store  *store.Store
	mux    *http.ServeMux
}

func newServer(cfg *config.Config) (*server, error) {
	db, err := store.Open(cfg.DataPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	s := &server{
		config: cfg,
		store:  db,
		mux:    http.NewServeMux(),
	}

	s.setupRoutes()
	return s, nil
}

func (s *server) setupRoutes() {
	fs := http.FileServer(http.Dir("./static"))
	s.mux.Handle("/static/", http.StripPrefix("/static/", fs))

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"inc": func(i int) int { return i + 1 },
	}

	shelfTemplate := template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/shelf.html",
	))

	viewerTemplate := template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/viewer.html",
	))

	overviewTemplate := template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/overview.html",
	))

	bibliographicTemplate := template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/bibliographic.html",
	))

	s.mux.Handle("/", &handlers.ShelfHandler{
		Store:    s.store,
		Template: shelfTemplate,
	})

	// Routes /books/{uuid}/overview, /books/{uuid}/bibliography,
	// and /books/{uuid}/pages/{page_num}.
	s.mux.Handle("/books/", &handlers.BookDispatchHandler{
		Store:                 s.store,
		OverviewTemplate:      overviewTemplate,
		BibliographicTemplate: bibliographicTemplate,
		ViewerTemplate:        viewerTemplate,
	})

	s.mux.Handle("/images/", &handlers.ImageHandler{
		Store: s.store,
	})

	s.mux.Handle("/thumbnails/", &handlers.ThumbnailHandler{
		Store: s.store,
	})

	// Handles PUT /api/books/{id}, PUT /api/books/{id}/note,
	// and POST /api/books/{id}/thumbnail.
	s.mux.Handle("/api/books/", &handlers.APIHandler{
		Store: s.store,
	})

	// Handles PUT /api/pages/{bookID}/{pageHash},
	// PUT /api/pages/{bookID}/{pageHash}/drawing, and
	// PUT /api/pages/{bookID}/{pageHash}/status.
	s.mux.Handle("/api/pages/", &handlers.NoteAPIHandler{
		Store: s.store,
	})

	cHandler := &handlers.CollectionsAPIHandler{Store: s.store}
	s.mux.Handle("/api/collections", cHandler)
	s.mux.Handle("/api/collections/", cHandler)

	s.mux.Handle("/page-thumbnails/", &handlers.ImageThumbnailHandler{
		Store: s.store,
	})
}

func (s *server) Start() error {
	addr := s.config.Address()
	fmt.Printf("Starting server at %s\n", addr)
	return http.ListenAndServe(addr, s.mux)
}

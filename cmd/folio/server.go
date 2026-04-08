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

	booksTemplate := template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/books.html",
	))

	viewerTemplate := template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/viewer.html",
	))

	s.mux.Handle("/", &handlers.BooksHandler{
		Store:    s.store,
		Template: booksTemplate,
	})

	s.mux.Handle("/viewer", &handlers.ViewerHandler{
		Store:    s.store,
		Template: viewerTemplate,
	})

	s.mux.Handle("/images/", &handlers.ImageHandler{
		Store: s.store,
	})

	s.mux.Handle("/api/books/", &handlers.APIHandler{
		Store: s.store,
	})
}

func (s *server) Start() error {
	addr := s.config.Address()
	fmt.Printf("Starting server at %s\n", addr)
	return http.ListenAndServe(addr, s.mux)
}

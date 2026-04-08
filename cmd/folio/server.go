package server

import (
	"fmt"
	"html/template"
	"net/http"

	"folio/internal/config"
	"folio/internal/handlers"
	"folio/internal/store"
)

type Server struct {
	Config *config.Config
	Store  *store.Store
	Mux    *http.ServeMux
}

func New(cfg *config.Config) (*Server, error) {
	db, err := store.Open(cfg.DataPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	s := &Server{
		Config: cfg,
		Store:  db,
		Mux:    http.NewServeMux(),
	}

	s.setupRoutes()
	return s, nil
}

func (s *Server) setupRoutes() {
	// Static files.
	fs := http.FileServer(http.Dir("./static"))
	s.Mux.Handle("/static/", http.StripPrefix("/static/", fs))

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

	s.Mux.Handle("/", &handlers.BooksHandler{
		Store:    s.Store,
		Template: booksTemplate,
	})

	s.Mux.Handle("/viewer", &handlers.ViewerHandler{
		Store:    s.Store,
		Template: viewerTemplate,
	})

	s.Mux.Handle("/images/", &handlers.ImageHandler{
		Store: s.Store,
	})
}

func (s *Server) Start() error {
	addr := s.Config.Address()
	fmt.Printf("Starting server at %s\n", addr)
	return http.ListenAndServe(addr, s.Mux)
}

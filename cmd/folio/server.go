package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"folio/internal/config"
	"folio/internal/handlers"
	"folio/internal/store"
)

//go:embed favicon.svg
var faviconSVG string

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

	s.mux.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(faviconSVG))
	})

	// Browsers automatically request /favicon.ico regardless of the <link> tag.
	// Without this handler the request falls through to HomeHandler and returns
	// an HTML 404, which Firefox caches — causing the favicon to disappear until
	// the cache is cleared. Serving the SVG here resolves the issue.
	s.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(faviconSVG))
	})

	fs := http.FileServer(http.Dir("./static"))
	s.mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Thumbnails are served directly from the cache directory via http.FileServer.
	// This gives ETag, Last-Modified, and conditional GET support for free.
	// URLs: /book-thumbnails/{bookID}.jpg  and  /page-thumbnails/{bookID}/{pageHash}.jpg
	bookThumbFS := http.FileServer(http.Dir(filepath.Join(s.config.CachePath, "book-thumbnails")))
	s.mux.Handle("/book-thumbnails/", http.StripPrefix("/book-thumbnails/", bookThumbFS))

	pageThumbFS := http.FileServer(http.Dir(filepath.Join(s.config.CachePath, "page-thumbnails")))
	s.mux.Handle("/page-thumbnails/", http.StripPrefix("/page-thumbnails/", pageThumbFS))

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"inc": func(i int) int { return i + 1 },
	}

	// sidebar.html is included in both home and collection template sets so
	// that the {{template "sidebar" .}} call resolves in both.
	homeTemplate := template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/sidebar.html",
		"templates/home.html",
	))

	collectionTemplate := template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/sidebar.html",
		"templates/collection.html",
	))

	uncategorizedTemplate := template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/sidebar.html",
		"templates/uncategorized.html",
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
		"templates/bibliography.html",
	))

	s.mux.Handle("/", &handlers.HomeHandler{
		Store:     s.store,
		CachePath: s.config.CachePath,
		Template:  homeTemplate,
	})

	s.mux.Handle("/collections/", &handlers.CollectionPageHandler{
		Store:     s.store,
		CachePath: s.config.CachePath,
		Template:  collectionTemplate,
	})

	// /books/uncategorized is registered as an exact (fixed) pattern so it
	// takes priority over the /books/ subtree pattern below.
	s.mux.Handle("/books/uncategorized", &handlers.UncategorizedPageHandler{
		Store:     s.store,
		CachePath: s.config.CachePath,
		Template:  uncategorizedTemplate,
	})

	// Routes /books/{uuid}/overview, /books/{uuid}/bibliography,
	// and /books/{uuid}/pages/{page_num}.
	s.mux.Handle("/books/", &handlers.BookDispatchHandler{
		Store:                 s.store,
		CachePath:             s.config.CachePath,
		OverviewTemplate:      overviewTemplate,
		BibliographicTemplate: bibliographicTemplate,
		ViewerTemplate:        viewerTemplate,
	})

	s.mux.Handle("/images/", &handlers.ImageHandler{
		Store: s.store,
	})

	// Handles PUT /api/books/{id}, PUT /api/books/{id}/note,
	// and POST /api/books/{id}/thumbnail.
	s.mux.Handle("/api/books/", &handlers.BooksAPIHandler{
		Store:     s.store,
		CachePath: s.config.CachePath,
	})

	// Handles PUT /api/pages/{bookID}/{pageHash},
	// PUT /api/pages/{bookID}/{pageHash}/drawing, and
	// PUT /api/pages/{bookID}/{pageHash}/status.
	s.mux.Handle("/api/pages/", &handlers.PagesAPIHandler{
		Store: s.store,
	})

	cHandler := &handlers.CollectionsAPIHandler{Store: s.store}
	s.mux.Handle("/api/collections", cHandler)
	s.mux.Handle("/api/collections/", cHandler)
}

func (s *server) Start() error {
	addr := s.config.Address()
	fmt.Printf("Starting server at %s\n", addr)
	return http.ListenAndServe(addr, s.mux)
}

func runServer(cfg *config.Config) {
	srv, err := newServer(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
}

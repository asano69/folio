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

	s.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(faviconSVG))
	})

	fs := http.FileServer(http.Dir("./static"))
	s.mux.Handle("/static/", http.StripPrefix("/static/", fs))

	bookThumbFS := http.FileServer(http.Dir(filepath.Join(s.config.CachePath, "book-thumbnails")))
	s.mux.Handle("/book-thumbnails/", http.StripPrefix("/book-thumbnails/", bookThumbFS))

	pageThumbFS := http.FileServer(http.Dir(filepath.Join(s.config.CachePath, "page-thumbnails")))
	s.mux.Handle("/page-thumbnails/", http.StripPrefix("/page-thumbnails/", pageThumbFS))

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"inc": func(i int) int { return i + 1 },
	}

	// Shared template for all book-listing pages (all, uncategorized, collections).
	collectionTemplate := template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/sidebar.html",
		"templates/collection.html",
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

	libraryTemplate := template.Must(template.New("layout.html").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/library.html",
	))

	// / → redirect to /collections/all
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/collections/all", http.StatusFound)
	})

	// Backward-compat redirect: /library → /libraries/all
	s.mux.HandleFunc("/library", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/libraries/all", http.StatusMovedPermanently)
	})

	// /collections/all — all books (exact match takes priority over /collections/ prefix)
	s.mux.Handle("/collections/all", &handlers.AllBooksHandler{
		Store:     s.store,
		CachePath: s.config.CachePath,
		Template:  collectionTemplate,
	})

	// /collections/uncategorized — books with no collection membership
	s.mux.Handle("/collections/uncategorized", &handlers.UncategorizedPageHandler{
		Store:     s.store,
		CachePath: s.config.CachePath,
		Template:  collectionTemplate,
	})

	// /collections/{uuid} — specific collection
	s.mux.Handle("/collections/", &handlers.CollectionPageHandler{
		Store:     s.store,
		CachePath: s.config.CachePath,
		Template:  collectionTemplate,
	})

	// /libraries/all — Central Library admin (exact match)
	libraryHandler := &handlers.LibraryPageHandler{
		Store:    s.store,
		Template: libraryTemplate,
	}
	s.mux.Handle("/libraries/all", libraryHandler)

	// /libraries/{uuid} — specific library admin
	s.mux.Handle("/libraries/", libraryHandler)

	// /books/{uuid}/... — viewer, overview, bibliography
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

	s.mux.Handle("/api/books/", &handlers.BooksAPIHandler{
		Store:     s.store,
		CachePath: s.config.CachePath,
	})

	s.mux.Handle("/api/pages/", &handlers.PagesAPIHandler{
		Store: s.store,
	})

	sectionsHandler := &handlers.SectionsAPIHandler{Store: s.store}
	s.mux.Handle("/api/sections", sectionsHandler)
	s.mux.Handle("/api/sections/", sectionsHandler)

	cHandler := &handlers.CollectionsAPIHandler{Store: s.store}
	s.mux.Handle("/api/collections", cHandler)
	s.mux.Handle("/api/collections/", cHandler)

	lHandler := &handlers.LibrariesAPIHandler{Store: s.store}
	s.mux.Handle("/api/libraries", lHandler)
	s.mux.Handle("/api/libraries/", lHandler)
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

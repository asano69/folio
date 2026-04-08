package server

import (
	"fmt"
	"html/template"
	"net/http"
	"openbook/internal/config"
	"openbook/internal/handlers"
	"openbook/internal/library"
)

type Server struct {
	Config  *config.Config
	Library *library.Library
	Mux     *http.ServeMux
}

func New(cfg *config.Config) (*Server, error) {
	// ライブラリをスキャン
	lib, err := library.NewLibrary(cfg.LibraryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create library: %w", err)
	}

	s := &Server{
		Config:  cfg,
		Library: lib,
		Mux:     http.NewServeMux(),
	}

	s.setupRoutes()
	return s, nil
}

func (s *Server) setupRoutes() {
	// 静的ファイル
	fs := http.FileServer(http.Dir("./static"))
	s.Mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// テンプレート関数
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"inc": func(i int) int { return i + 1 },
	}

	// テンプレート読み込み
	booksTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/books.html",
	))

	viewerTemplate := template.Must(template.New("").Funcs(funcMap).ParseFiles(
		"templates/layout.html",
		"templates/viewer.html",
	))

	// ハンドラ
	s.Mux.Handle("/", &handlers.BooksHandler{
		Library:  s.Library,
		Template: booksTemplate,
	})

	s.Mux.Handle("/viewer", &handlers.ViewerHandler{
		Library:  s.Library,
		Template: viewerTemplate,
	})

	s.Mux.Handle("/images/", &handlers.ImageHandler{
		Library: s.Library,
	})
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port)
	fmt.Printf("Starting server at %s\n", addr)
	fmt.Printf("Library path: %s\n", s.Config.LibraryPath)
	fmt.Printf("Found %d books\n", len(s.Library.Books))

	return http.ListenAndServe(addr, s.Mux)
}

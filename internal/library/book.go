package library

type Book struct {
	ID    string
	Title string
	Pages []Page
}

type Page struct {
	Number   int
	Filename string
	Path     string
}

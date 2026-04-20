package storage

// PersonName represents a person's name split into family and given components,
// following the CSL (Citation Style Language) convention.
type PersonName struct {
	Family string `json:"family"`
	Given  string `json:"given"`
}

// Book represents a scanned CBZ file with all metadata extracted from folio.json.
type Book struct {
	ID        string
	Title     string
	Source    string // absolute path to the CBZ file
	FileMtime int64  // Unix timestamp from os.Stat; used to detect CBZ changes between scans
	Pages     []ImageEntry
	// Optional metadata mirrored from folio.json
	Type         string
	Abstract     string
	Language     string
	Author       []PersonName
	Translator   []PersonName
	OrigTitle    string
	Edition      string
	Volume       string
	Series       string
	SeriesNumber string
	Publisher    string
	Year         string
	Note         string
	Keywords     []string
	ISBN         string
	Links        []string
}

// ImageEntry represents a single image entry inside a CBZ.
type ImageEntry struct {
	Seq      int    // 1-based position within the CBZ (filename sort order)
	Filename string // entry name inside the CBZ
	Hash     string // SHA-256 of the uncompressed image bytes
}

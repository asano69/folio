package storage

// Book represents a scanned CBZ file.
type Book struct {
	ID        string
	Title     string
	Source    string // absolute path to the CBZ file
	FileMtime int64  // Unix timestamp from os.Stat; used to detect CBZ changes between scans
	Pages     []ImageEntry
}

// ImageEntry represents a single image entry inside a CBZ.
type ImageEntry struct {
	Number   int
	Filename string // entry name inside the CBZ
	Hash     string // SHA-256 of the uncompressed image bytes
}

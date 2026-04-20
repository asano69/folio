package storage

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	metaFile    = "folio.json"
	metaVersion = "2026-04-20"
)

// folioMeta is the in-memory representation of folio.json stored inside a CBZ.
// Fields map directly to the JSON keys in the file.
// Required fields: version, id, title.
// All other fields are optional (omitempty).
type folioMeta struct {
	Version      string       `json:"version"`
	ID           string       `json:"id"`
	Type         string       `json:"type,omitempty"`
	Abstract     string       `json:"abstract,omitempty"`
	Language     string       `json:"language,omitempty"`
	Author       []PersonName `json:"author,omitempty"`
	Translator   []PersonName `json:"translator,omitempty"`
	Title        string       `json:"title"`
	OrigTitle    string       `json:"origtitle,omitempty"`
	Edition      string       `json:"edition,omitempty"`
	Volume       string       `json:"volume,omitempty"`
	Series       string       `json:"series,omitempty"`
	SeriesNumber string       `json:"series_number,omitempty"`
	Publisher    string       `json:"publisher,omitempty"`
	Year         string       `json:"year,omitempty"`
	Note         string       `json:"note,omitempty"`
	Keywords     []string     `json:"keywords,omitempty"`
	ISBN         string       `json:"isbn,omitempty"`
	Links        []string     `json:"links,omitempty"`
	CreatedAt    string       `json:"created_at,omitempty"`
	UpdatedAt    string       `json:"updated_at,omitempty"`
}

// metaToBook converts a folioMeta into a Book, setting source and mtime from the caller.
func metaToBook(meta *folioMeta, source string, mtime int64) Book {
	return Book{
		ID:           meta.ID,
		Title:        meta.Title,
		Source:       source,
		FileMtime:    mtime,
		Type:         meta.Type,
		Abstract:     meta.Abstract,
		Language:     meta.Language,
		Author:       meta.Author,
		Translator:   meta.Translator,
		OrigTitle:    meta.OrigTitle,
		Edition:      meta.Edition,
		Volume:       meta.Volume,
		Series:       meta.Series,
		SeriesNumber: meta.SeriesNumber,
		Publisher:    meta.Publisher,
		Year:         meta.Year,
		Note:         meta.Note,
		Keywords:     meta.Keywords,
		ISBN:         meta.ISBN,
		Links:        meta.Links,
	}
}

// bookToMeta converts a Book's metadata fields into a folioMeta.
// Timestamps (created_at, updated_at) and version must be set by the caller.
func bookToMeta(b Book) *folioMeta {
	return &folioMeta{
		ID:           b.ID,
		Type:         b.Type,
		Abstract:     b.Abstract,
		Language:     b.Language,
		Author:       b.Author,
		Translator:   b.Translator,
		Title:        b.Title,
		OrigTitle:    b.OrigTitle,
		Edition:      b.Edition,
		Volume:       b.Volume,
		Series:       b.Series,
		SeriesNumber: b.SeriesNumber,
		Publisher:    b.Publisher,
		Year:         b.Year,
		Note:         b.Note,
		Keywords:     b.Keywords,
		ISBN:         b.ISBN,
		Links:        b.Links,
	}
}

// openCBZMeta reads only folio.json from a CBZ and returns book identity and
// the file's modification time. Images are not listed and hashes are not computed,
// making this significantly faster than openCBZ for already-registered books.
//
// If folio.json is absent, the returned Book has an empty ID, signalling that a
// full open via openCBZ is required to generate one.
func openCBZMeta(path string) (Book, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return Book{}, fmt.Errorf("open cbz %s: %w", path, err)
	}
	defer r.Close()

	meta, err := readMeta(r)
	if err != nil {
		return Book{}, err
	}

	info, err := os.Stat(path)
	if err != nil {
		return Book{}, fmt.Errorf("stat %s: %w", path, err)
	}

	if meta == nil {
		// No folio.json yet; full open will generate one.
		return Book{Source: path, FileMtime: info.ModTime().Unix()}, nil
	}

	return metaToBook(meta, path, info.ModTime().Unix()), nil
}

func openCBZ(path string) (Book, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return Book{}, fmt.Errorf("open cbz %s: %w", path, err)
	}

	meta, err := readMeta(r)
	if err != nil {
		r.Close()
		return Book{}, err
	}

	// If no folio.json found, generate one and write it back.
	// writeMeta closes r before overwriting the file (ZIP structure requires a full rewrite),
	// so we must reopen the archive afterwards to read image entries.
	if meta == nil {
		id, err := uuid.NewV7()
		if err != nil {
			r.Close()
			return Book{}, fmt.Errorf("generate uuid: %w", err)
		}
		now := time.Now().Format(time.RFC3339)
		m := &folioMeta{
			Version:   metaVersion,
			ID:        id.String(),
			Title:     strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
			CreatedAt: now,
			UpdatedAt: now,
		}
		// writeMeta closes r as a side effect.
		if err := writeMeta(path, r, m); err != nil {
			return Book{}, fmt.Errorf("write folio.json: %w", err)
		}
		meta = m

		// Reopen after the rewrite so listImages can read the updated archive.
		r, err = zip.OpenReader(path)
		if err != nil {
			return Book{}, fmt.Errorf("reopen cbz after write %s: %w", path, err)
		}
	}
	defer r.Close()

	images, err := listImages(r)
	if err != nil {
		return Book{}, err
	}

	// Stat after any potential writeMeta call so FileMtime reflects the final
	// state of the file on disk.
	info, err := os.Stat(path)
	if err != nil {
		return Book{}, fmt.Errorf("stat %s: %w", path, err)
	}

	book := metaToBook(meta, path, info.ModTime().Unix())
	book.Pages = images
	return book, nil
}

// OpenBook opens a single CBZ file and returns its book data including images
// with hashes. It is the exported equivalent of openCBZ, intended for use by
// commands that operate on one book at a time (e.g. "folio hash <uuid>").
func OpenBook(path string) (Book, error) {
	return openCBZ(path)
}

// OpenPage returns a reader for a single image inside a CBZ.
func OpenPage(cbzPath, filename string) (io.ReadCloser, error) {
	r, err := zip.OpenReader(cbzPath)
	if err != nil {
		return nil, fmt.Errorf("open cbz %s: %w", cbzPath, err)
	}

	for _, f := range r.File {
		if f.Name == filename {
			rc, err := f.Open()
			if err != nil {
				r.Close()
				return nil, err
			}
			// Wrap so closing also closes the zip reader.
			return &pageReader{rc: rc, zip: r}, nil
		}
	}

	r.Close()
	return nil, fmt.Errorf("page %s not found in %s", filename, cbzPath)
}

// UpdateTitle rewrites only the title field in folio.json inside the CBZ archive.
// This is used by the inline rename on the library page.
// For full metadata updates, use UpdateFolioMeta instead.
func UpdateTitle(cbzPath, newTitle string) error {
	r, err := zip.OpenReader(cbzPath)
	if err != nil {
		return fmt.Errorf("open cbz %s: %w", cbzPath, err)
	}

	meta, err := readMeta(r)
	if err != nil {
		r.Close()
		return err
	}
	if meta == nil {
		r.Close()
		return fmt.Errorf("folio.json not found in %s", cbzPath)
	}

	meta.Title = newTitle
	meta.Version = metaVersion
	meta.UpdatedAt = time.Now().Format(time.RFC3339)
	// writeMeta closes r before overwriting the file.
	return writeMeta(cbzPath, r, meta)
}

// UpdateFolioMeta rewrites folio.json inside the CBZ with the given book metadata.
// The existing created_at timestamp is preserved; updated_at is set to the current time.
// This is used by the bibliography page metadata editor.
func UpdateFolioMeta(cbzPath string, b Book) error {
	r, err := zip.OpenReader(cbzPath)
	if err != nil {
		return fmt.Errorf("open cbz %s: %w", cbzPath, err)
	}

	existing, err := readMeta(r)
	if err != nil {
		r.Close()
		return err
	}

	meta := bookToMeta(b)
	meta.Version = metaVersion
	meta.UpdatedAt = time.Now().Format(time.RFC3339)

	if existing != nil && existing.CreatedAt != "" {
		meta.CreatedAt = existing.CreatedAt
	} else {
		// Preserve created_at as the current time if no prior value exists.
		meta.CreatedAt = meta.UpdatedAt
	}

	// writeMeta closes r before overwriting the file.
	return writeMeta(cbzPath, r, meta)
}

type pageReader struct {
	rc  io.ReadCloser
	zip *zip.ReadCloser
}

func (p *pageReader) Read(b []byte) (int, error) { return p.rc.Read(b) }
func (p *pageReader) Close() error {
	p.rc.Close()
	return p.zip.Close()
}

// readMeta reads folio.json from an open zip, returns nil if not present.
func readMeta(r *zip.ReadCloser) (*folioMeta, error) {
	for _, f := range r.File {
		if f.Name != metaFile {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()

		var m folioMeta
		if err := json.NewDecoder(rc).Decode(&m); err != nil {
			return nil, fmt.Errorf("decode folio.json: %w", err)
		}
		return &m, nil
	}
	return nil, nil
}

// writeMeta rewrites the CBZ with folio.json added or replaced.
// Existing entries are copied as raw compressed bytes to preserve their CRC32 values.
// writeMeta closes r before overwriting the file.
func writeMeta(path string, r *zip.ReadCloser, meta *folioMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	// Build new zip in memory, then overwrite the file.
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	// Copy existing entries as raw bytes to preserve original CRC32 values.
	for _, f := range r.File {
		if f.Name == metaFile {
			continue // will be replaced
		}
		fw, err := w.CreateRaw(&f.FileHeader)
		if err != nil {
			return err
		}
		rc, err := f.OpenRaw()
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(fw, rc)
		if copyErr != nil {
			return copyErr
		}
	}

	// Write folio.json.
	fw, err := w.Create(metaFile)
	if err != nil {
		return err
	}
	if _, err := fw.Write(data); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		return err
	}

	// Close the reader before overwriting the file.
	r.Close()

	return os.WriteFile(path, buf.Bytes(), 0644)
}

// listImages returns image entries from an open zip, sorted by filename.
// Each entry's Hash is computed as the SHA-256 of its uncompressed bytes.
// Seq is assigned as the 1-based position in the sorted order.
func listImages(r *zip.ReadCloser) ([]ImageEntry, error) {
	var images []ImageEntry
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue
		}

		hash, err := hashEntry(f)
		if err != nil {
			return nil, fmt.Errorf("hash %s: %w", f.Name, err)
		}

		images = append(images, ImageEntry{Filename: f.Name, Hash: hash})
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].Filename < images[j].Filename
	})

	for i := range images {
		images[i].Seq = i + 1
	}

	return images, nil
}

// hashEntry computes the SHA-256 of a zip entry's uncompressed bytes.
func hashEntry(f *zip.File) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	h := sha256.New()
	if _, err := io.Copy(h, rc); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

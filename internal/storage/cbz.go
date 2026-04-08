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

	"github.com/google/uuid"
)

const metaFile = "folio.json"

type folioMeta struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// openCBZ opens a CBZ file and returns its metadata and page list.
// If folio.json is missing, a new UUID v7 is generated and written into the archive.
func openCBZ(path string) (Book, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return Book{}, fmt.Errorf("open cbz %s: %w", path, err)
	}
	defer r.Close()

	meta, err := readMeta(r)
	if err != nil {
		return Book{}, err
	}

	// If no folio.json found, generate one and write it back.
	if meta == nil {
		id, err := uuid.NewV7()
		if err != nil {
			return Book{}, fmt.Errorf("generate uuid: %w", err)
		}
		m := &folioMeta{
			ID:    id.String(),
			Title: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		}
		if err := writeMeta(path, r, m); err != nil {
			return Book{}, fmt.Errorf("write folio.json: %w", err)
		}
		meta = m
	}

	pages, err := listPages(r)
	if err != nil {
		return Book{}, err
	}

	return Book{
		ID:     meta.ID,
		Title:  meta.Title,
		Source: path,
		Pages:  pages,
	}, nil
}

// OpenPage returns a reader for a single page inside a CBZ.
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

// UpdateTitle rewrites the title field in folio.json inside the CBZ archive.
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

// writeMeta rewrites the CBZ with folio.json added.
// Existing entries are copied as raw compressed bytes to preserve their CRC32 values.
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

// listPages returns image entries from an open zip, sorted by filename.
// Each page's Hash is computed as the SHA-256 of its uncompressed bytes.
func listPages(r *zip.ReadCloser) ([]Page, error) {
	var pages []Page
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

		pages = append(pages, Page{Filename: f.Name, Hash: hash})
	}

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Filename < pages[j].Filename
	})

	for i := range pages {
		pages[i].Number = i + 1
	}

	return pages, nil
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

package storage

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/image/draw"
)

const (
	bookThumbnailWidth = 400
	pageThumbnailWidth = 300
)

// GenerateBookThumbnail opens the first image in a CBZ and returns a
// JPEG-encoded thumbnail scaled to bookThumbnailWidth pixels wide.
func GenerateBookThumbnail(cbzPath string) ([]byte, error) {
	r, err := zip.OpenReader(cbzPath)
	if err != nil {
		return nil, fmt.Errorf("open cbz %s: %w", cbzPath, err)
	}
	defer r.Close()

	images, err := listImages(r)
	if err != nil {
		return nil, fmt.Errorf("list images %s: %w", cbzPath, err)
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("no images in %s", cbzPath)
	}

	first := images[0].Filename
	for _, f := range r.File {
		if f.Name == first {
			return thumbnailFromEntry(f, bookThumbnailWidth)
		}
	}
	return nil, fmt.Errorf("image entry %s not found in %s", first, cbzPath)
}

// ImageThumbnailRequest pairs a page ID with its filename for thumbnail generation.
// PageID is the stable integer primary key from the pages table.
type ImageThumbnailRequest struct {
	PageID   int
	Filename string
}

// ImageThumbnailResult holds the generated thumbnail data for one page.
type ImageThumbnailResult struct {
	PageID int
	Data   []byte
}

// GeneratePageThumbnails opens cbzPath once and generates JPEG thumbnails for
// every requested page. Opening the ZIP once amortises the central-directory
// read cost across all pages in the batch.
// Pages whose filename is not found in the archive are silently skipped.
func GeneratePageThumbnails(cbzPath string, reqs []ImageThumbnailRequest) ([]ImageThumbnailResult, error) {
	r, err := zip.OpenReader(cbzPath)
	if err != nil {
		return nil, fmt.Errorf("open cbz %s: %w", cbzPath, err)
	}
	defer r.Close()

	// Index requests by filename for O(1) lookup while iterating the archive.
	need := make(map[string]int, len(reqs)) // filename -> pageID
	for _, req := range reqs {
		need[req.Filename] = req.PageID
	}

	var results []ImageThumbnailResult
	for _, f := range r.File {
		pageID, ok := need[f.Name]
		if !ok {
			continue
		}
		data, err := thumbnailFromEntry(f, pageThumbnailWidth)
		if err != nil {
			return nil, fmt.Errorf("thumbnail %s: %w", f.Name, err)
		}
		results = append(results, ImageThumbnailResult{PageID: pageID, Data: data})
	}
	return results, nil
}

// thumbnailFromEntry decodes and resizes a single ZIP entry into a JPEG thumbnail.
func thumbnailFromEntry(f *zip.File, width int) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	src, _, err := image.Decode(rc)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resizeToWidth(src, width), &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return buf.Bytes(), nil
}

// resizeToWidth scales img proportionally so its width equals w.
// ApproxBiLinear is used instead of BiLinear: quality is indistinguishable at
// thumbnail sizes and performance is significantly better.
func resizeToWidth(src image.Image, w int) image.Image {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW == 0 {
		return src
	}
	h := w * srcH / srcW
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

// ── Filesystem cache helpers ───────────────────────────────────
//
// Book thumbnails:  {cachePath}/book-thumbnails/{bookID}.jpg
// Page thumbnails:  {cachePath}/page-thumbnails/{bookID}/{pageID}.jpg
//
// Page thumbnails are keyed by stable integer page ID rather than content hash,
// so they survive re-scans and in-place image replacements.

// BookThumbnailPath returns the filesystem path for a book-level thumbnail.
func BookThumbnailPath(cachePath, bookID string) string {
	return filepath.Join(cachePath, "book-thumbnails", bookID+".jpg")
}

// PageThumbnailPath returns the filesystem path for a page-level thumbnail.
// Thumbnails are organised under a per-book subdirectory to keep the
// page-thumbnails directory manageable.
func PageThumbnailPath(cachePath, bookID string, pageID int) string {
	return filepath.Join(cachePath, "page-thumbnails", bookID, strconv.Itoa(pageID)+".jpg")
}

// WriteBookThumbnail writes book-level thumbnail bytes to the cache directory,
// creating any missing parent directories.
func WriteBookThumbnail(cachePath, bookID string, data []byte) error {
	path := BookThumbnailPath(cachePath, bookID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create book-thumbnails dir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// WritePageThumbnail writes a page-level thumbnail to the cache directory,
// creating any missing parent directories.
func WritePageThumbnail(cachePath, bookID string, pageID int, data []byte) error {
	path := PageThumbnailPath(cachePath, bookID, pageID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create page-thumbnails dir: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// BookThumbnailExists reports whether a cached thumbnail file exists for the given book.
func BookThumbnailExists(cachePath, bookID string) bool {
	_, err := os.Stat(BookThumbnailPath(cachePath, bookID))
	return err == nil
}

// PageThumbnailExists reports whether a cached thumbnail file exists for the given page.
func PageThumbnailExists(cachePath, bookID string, pageID int) bool {
	_, err := os.Stat(PageThumbnailPath(cachePath, bookID, pageID))
	return err == nil
}

// ListBookThumbnailIDs returns the set of book IDs that have a cached
// book-level thumbnail. Returns an empty map (not an error) when the directory
// does not yet exist.
func ListBookThumbnailIDs(cachePath string) (map[string]bool, error) {
	dir := filepath.Join(cachePath, "book-thumbnails")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read book-thumbnails dir: %w", err)
	}
	set := make(map[string]bool, len(entries))
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".jpg") {
			set[strings.TrimSuffix(name, ".jpg")] = true
		}
	}
	return set, nil
}

// ListPageThumbnailIDs returns the set of page IDs (integers) that have a
// cached page-level thumbnail for the given book. Returns an empty map (not
// an error) when the directory does not yet exist.
func ListPageThumbnailIDs(cachePath, bookID string) (map[int]bool, error) {
	dir := filepath.Join(cachePath, "page-thumbnails", bookID)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return map[int]bool{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read page-thumbnails dir: %w", err)
	}
	set := make(map[int]bool, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".jpg") {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSuffix(name, ".jpg"))
		if err != nil {
			continue // skip files that are not integer-named
		}
		set[id] = true
	}
	return set, nil
}

package storage

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
)

const thumbnailWidth = 200

// GenerateThumbnail opens the first image page in a CBZ and returns a
// JPEG-encoded thumbnail scaled to thumbnailWidth pixels wide.
func GenerateThumbnail(cbzPath string) ([]byte, error) {
	r, err := zip.OpenReader(cbzPath)
	if err != nil {
		return nil, fmt.Errorf("open cbz %s: %w", cbzPath, err)
	}
	defer r.Close()

	pages, err := listPages(r)
	if err != nil {
		return nil, fmt.Errorf("list pages %s: %w", cbzPath, err)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no image pages in %s", cbzPath)
	}

	first := pages[0].Filename
	for _, f := range r.File {
		if f.Name == first {
			return thumbnailFromEntry(f)
		}
	}
	return nil, fmt.Errorf("page entry %s not found in %s", first, cbzPath)
}

// PageThumbnailRequest pairs a page filename with its content hash.
// Hash is carried through so callers can key the result without re-deriving it.
type PageThumbnailRequest struct {
	Filename string
	Hash     string
}

// PageThumbnailResult holds the generated thumbnail for one page.
type PageThumbnailResult struct {
	Hash string
	Data []byte
}

// GeneratePageThumbnails opens cbzPath once and generates JPEG thumbnails for
// every requested page. Pages not found in the archive are silently skipped.
//
// Opening the ZIP once amortises the cost of reading the central directory
// (stored at the end of the file) across all pages, which is significantly
// faster than calling a per-page function in a loop.
func GeneratePageThumbnails(cbzPath string, reqs []PageThumbnailRequest) ([]PageThumbnailResult, error) {
	r, err := zip.OpenReader(cbzPath)
	if err != nil {
		return nil, fmt.Errorf("open cbz %s: %w", cbzPath, err)
	}
	defer r.Close()

	// Index requests by filename for O(1) lookup while iterating the archive.
	need := make(map[string]string, len(reqs))
	for _, req := range reqs {
		need[req.Filename] = req.Hash
	}

	var results []PageThumbnailResult
	for _, f := range r.File {
		hash, ok := need[f.Name]
		if !ok {
			continue
		}
		data, err := thumbnailFromEntry(f)
		if err != nil {
			return nil, fmt.Errorf("thumbnail %s: %w", f.Name, err)
		}
		results = append(results, PageThumbnailResult{Hash: hash, Data: data})
	}
	return results, nil
}

// thumbnailFromEntry decodes and resizes a single ZIP entry into a JPEG thumbnail.
func thumbnailFromEntry(f *zip.File) ([]byte, error) {
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
	if err := jpeg.Encode(&buf, resizeToWidth(src, thumbnailWidth), &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return buf.Bytes(), nil
}

// resizeToWidth scales img proportionally so its width equals w.
// ApproxBiLinear is used in place of BiLinear: quality is indistinguishable at
// thumbnail sizes and the performance is significantly better.
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

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

	return generateThumbnailFromEntry(r, pages[0].Filename, cbzPath)
}

// GeneratePageThumbnail opens the named image entry in a CBZ and returns a
// JPEG-encoded thumbnail scaled to thumbnailWidth pixels wide.
func GeneratePageThumbnail(cbzPath, filename string) ([]byte, error) {
	r, err := zip.OpenReader(cbzPath)
	if err != nil {
		return nil, fmt.Errorf("open cbz %s: %w", cbzPath, err)
	}
	defer r.Close()

	return generateThumbnailFromEntry(r, filename, cbzPath)
}

// generateThumbnailFromEntry finds filename inside an open zip and returns a
// JPEG thumbnail. Extracted as a shared helper to avoid duplicating the
// decode/resize/encode pipeline.
func generateThumbnailFromEntry(r *zip.ReadCloser, filename, cbzPath string) ([]byte, error) {
	for _, f := range r.File {
		if f.Name != filename {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open page %s: %w", filename, err)
		}
		defer rc.Close()

		src, _, err := image.Decode(rc)
		if err != nil {
			return nil, fmt.Errorf("decode image %s: %w", filename, err)
		}

		thumb := resizeToWidth(src, thumbnailWidth)

		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, thumb, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("encode thumbnail: %w", err)
		}
		return buf.Bytes(), nil
	}

	return nil, fmt.Errorf("page entry %s not found in %s", filename, cbzPath)
}

// resizeToWidth scales img proportionally so its width equals w.
func resizeToWidth(src image.Image, w int) image.Image {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if srcW == 0 {
		return src
	}

	h := w * srcH / srcW
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}

// Package media extracts metadata from media files (images, videos).
//
// EXIF extraction is best-effort: if the file contains no EXIF data or
// the data is malformed, a nil *domain.MediaMeta is returned with no error.
// This keeps uploads working for all file types without requiring valid metadata.
package media

import (
	"io"
	"math"
	"strings"
	"time"

	"minicloud/internal/domain"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

// ExtractMetadata reads EXIF data from r if the MIME type is an image.
// Returns nil (not an error) when the file has no extractable metadata.
func ExtractMetadata(r io.Reader, mimeType string) *domain.MediaMeta {
	if !isExifCapable(mimeType) {
		return nil
	}

	x, err := exif.Decode(r)
	if err != nil {
		return nil
	}

	meta := &domain.MediaMeta{}
	populated := false

	// Date taken
	if t, err := x.DateTime(); err == nil {
		utc := t.UTC()
		meta.TakenAt = &utc
		populated = true
	}

	// Camera info
	if tag, err := x.Get(exif.Make); err == nil {
		meta.CameraMake = cleanString(tag)
		populated = true
	}
	if tag, err := x.Get(exif.Model); err == nil {
		meta.CameraModel = cleanString(tag)
		populated = true
	}

	// Dimensions
	if w, h, err := dimensions(x); err == nil {
		meta.Width = w
		meta.Height = h
		populated = true
	}

	// GPS
	if lat, lon, err := x.LatLong(); err == nil && !math.IsNaN(lat) && !math.IsNaN(lon) {
		meta.Latitude = &lat
		meta.Longitude = &lon
		populated = true
	}

	if !populated {
		return nil
	}
	return meta
}

// ParseTakenAtMonth returns the year and month from a taken_at time,
// formatted as "January 2006". Used for grouping media by month.
func ParseTakenAtMonth(t time.Time) string {
	return t.Format("January 2006")
}

// isExifCapable returns true for MIME types that commonly contain EXIF data.
func isExifCapable(mime string) bool {
	switch {
	case strings.HasPrefix(mime, "image/jpeg"),
		strings.HasPrefix(mime, "image/tiff"),
		strings.HasPrefix(mime, "image/heic"),
		strings.HasPrefix(mime, "image/heif"),
		strings.HasPrefix(mime, "image/webp"):
		return true
	}
	return false
}

// cleanString extracts a sanitized string value from a TIFF tag.
func cleanString(tag *tiff.Tag) string {
	s := strings.TrimSpace(tag.String())
	// exif lib returns quoted strings
	s = strings.Trim(s, "\"")
	return s
}

// dimensions extracts pixel width and height from EXIF data.
func dimensions(x *exif.Exif) (int, int, error) {
	wTag, err := x.Get(exif.PixelXDimension)
	if err != nil {
		return 0, 0, err
	}
	hTag, err := x.Get(exif.PixelYDimension)
	if err != nil {
		return 0, 0, err
	}
	w, err := wTag.Int(0)
	if err != nil {
		return 0, 0, err
	}
	h, err := hTag.Int(0)
	if err != nil {
		return 0, 0, err
	}
	return w, h, nil
}

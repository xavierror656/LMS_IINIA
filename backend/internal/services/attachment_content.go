package services

import (
	"aulaquest/internal/models"
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxAttachmentBytes = 2 * 1024 * 1024
const MaxAttachmentAccountBytes = 50 * 1024 * 1024

var ErrAttachmentTooLarge = errors.New("attachment exceeds size limit")
var ErrAttachmentBusy = errors.New("attachment processor busy")

// A process-wide resource bound; never stores user data between requests.
var imageSlots = make(chan struct{}, 2)

func NormalizeAttachment(name string, data []byte) ([]byte, string, error) {
	if !models.ValidText(name, 1, 120) || name != strings.TrimSpace(name) || strings.ContainsAny(name, "/\\") {
		return nil, "", models.ErrAcademicInput
	}
	for _, r := range name {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return nil, "", models.ErrAcademicInput
		}
	}
	if len(data) > MaxAttachmentBytes {
		return nil, "", ErrAttachmentTooLarge
	}
	if len(data) == 0 {
		return nil, "", models.ErrAcademicInput
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".txt" {
		if !utf8.Valid(data) {
			return nil, "", models.ErrAcademicInput
		}
		for _, r := range string(data) {
			if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
				return nil, "", models.ErrAcademicInput
			}
		}
		return data, "text/plain; charset=utf-8", nil
	}
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return nil, "", models.ErrAcademicInput
	}
	select {
	case imageSlots <- struct{}{}:
		defer func() { <-imageSlots }()
	default:
		return nil, "", ErrAttachmentBusy
	}
	cfg, format, e := image.DecodeConfig(bytes.NewReader(data))
	if e != nil {
		return nil, "", models.ErrAcademicInput
	}
	if (format != "png" && format != "jpeg") || (ext == ".png" && format != "png") || (ext != ".png" && format != "jpeg") {
		return nil, "", models.ErrAcademicInput
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 4_000_000/cfg.Height {
		return nil, "", ErrAttachmentTooLarge
	}
	img, _, e := image.Decode(bytes.NewReader(data))
	if e != nil {
		return nil, "", models.ErrAcademicInput
	}
	var out bytes.Buffer
	if format == "png" {
		e = png.Encode(&out, img)
	} else {
		e = jpeg.Encode(&out, img, &jpeg.Options{Quality: 85})
	}
	if e != nil {
		return nil, "", models.ErrAcademicInput
	}
	if out.Len() > MaxAttachmentBytes {
		return nil, "", ErrAttachmentTooLarge
	}
	return out.Bytes(), "image/" + format, nil
}

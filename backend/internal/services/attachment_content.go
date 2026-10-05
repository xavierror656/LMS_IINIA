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
	if ext == ".pdf" {
		if e := analyzePDF(data); e != nil {
			return nil, "", e
		}
		return data, "application/pdf", nil
	}
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

const maxPDFPages = 200

// pdfNameMarkers are the declared features a school worksheet never needs and that
// this screen refuses: encryption and active or embedded content.
var pdfNameMarkers = [][]byte{
	[]byte("/Encrypt"),
	[]byte("/JavaScript"),
	[]byte("/JS"),
	[]byte("/Launch"),
	[]byte("/RichMedia"),
	[]byte("/EmbeddedFile"),
}

// analyzePDF checks the structure of the file instead of trusting its extension:
// header, cross-reference table, trailer, page count and refusal of declared
// encryption or active content. It is a screen, not a sanitizer: object streams are
// compressed, so a marker can hide inside one and the search cannot prove absence.
// That is why a PDF is always delivered as a download and never inline.
func analyzePDF(data []byte) error {
	if !bytes.HasPrefix(data, []byte("%PDF-1.")) || len(data) < 64 {
		return models.ErrAcademicInput
	}
	tail := data
	if len(tail) > 4096 {
		tail = tail[len(tail)-4096:]
	}
	if !bytes.Contains(tail, []byte("%%EOF")) {
		return models.ErrAcademicInput
	}
	if !bytes.Contains(data, []byte("xref")) && !bytes.Contains(data, []byte("/XRef")) {
		return models.ErrAcademicInput
	}
	for _, marker := range pdfNameMarkers {
		if hasPDFName(data, marker) {
			return models.ErrAcademicInput
		}
	}
	if pages := pdfPages(data); pages == 0 || pages > maxPDFPages {
		return ErrAttachmentTooLarge
	}
	return nil
}

// hasPDFName reports whether the document declares that name, requiring a delimiter
// after it so a longer name or an ordinary string does not match.
func hasPDFName(data, name []byte) bool {
	for from := 0; ; {
		i := bytes.Index(data[from:], name)
		if i < 0 {
			return false
		}
		end := from + i + len(name)
		if end >= len(data) || pdfDelimiter(data[end]) {
			return true
		}
		from = end
	}
}
func pdfDelimiter(b byte) bool {
	switch b {
	case ' ', '\t', '\r', '\n', '\f', 0, '/', '[', ']', '<', '>', '(', ')', '%':
		return true
	}
	return false
}

// pdfPages counts page objects, tolerating the whitespace the specification allows
// between the type keyword and its value. /Type /Pages does not count as a page.
func pdfPages(data []byte) int {
	count := 0
	for i := 0; i+7 < len(data); i++ {
		if !bytes.HasPrefix(data[i:], []byte("/Type")) {
			continue
		}
		rest := data[i+5:]
		j := 0
		for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t' || rest[j] == '\r' || rest[j] == '\n' || rest[j] == '\f' || rest[j] == 0) {
			j++
		}
		if j >= len(rest) || !bytes.HasPrefix(rest[j:], []byte("/Page")) {
			continue
		}
		if len(rest) > j+5 && !pdfDelimiter(rest[j+5]) {
			continue
		}
		count++
	}
	return count
}

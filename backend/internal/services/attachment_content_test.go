package services

import (
	"aulaquest/internal/models"
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestAttachmentContent(t *testing.T) {
	for _, name := range []string{"../a.txt", "a\\b.txt", "a\r\nb.txt", "a\u202etxt.txt", "a.html", "a.svg", "a.pdf", "a.zip", " a.txt"} {
		if _, _, e := NormalizeAttachment(name, []byte("hola")); e == nil {
			t.Fatal("accepted", name)
		}
	}
	if _, _, e := NormalizeAttachment("a.txt", []byte{0xff}); !errors.Is(e, models.ErrAcademicInput) {
		t.Fatal(e)
	}
	if _, _, e := NormalizeAttachment("a.txt", []byte{'a', 0}); e == nil {
		t.Fatal("NUL accepted")
	}
	if _, _, e := NormalizeAttachment("a.txt", bytes.Repeat([]byte{'a'}, MaxAttachmentBytes+1)); !errors.Is(e, ErrAttachmentTooLarge) {
		t.Fatal(e)
	}
	data, kind, e := NormalizeAttachment("idea.txt", []byte("¡Mi idea!\n"))
	if e != nil || kind != "text/plain; charset=utf-8" || string(data) != "¡Mi idea!\n" {
		t.Fatal(kind, e)
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var input bytes.Buffer
	if e = png.Encode(&input, img); e != nil {
		t.Fatal(e)
	}
	raw := append(append([]byte{}, input.Bytes()...), []byte("PRIVATE_METADATA_TRAILER")...)
	data, kind, e = NormalizeAttachment("dibujo.png", raw)
	if e != nil || kind != "image/png" || bytes.Contains(data, []byte("PRIVATE_METADATA_TRAILER")) {
		t.Fatal("normalization", kind, e)
	}
	if _, _, e = NormalizeAttachment("fake.jpg", input.Bytes()); e == nil {
		t.Fatal("mismatched extension")
	}
	if _, _, e = NormalizeAttachment("fake.png", []byte("not an image")); e == nil {
		t.Fatal("fake image")
	}
	var large bytes.Buffer
	if e = png.Encode(&large, image.NewGray(image.Rect(0, 0, 2001, 2000))); e != nil {
		t.Fatal(e)
	}
	if _, _, e = NormalizeAttachment("large.png", large.Bytes()); !errors.Is(e, ErrAttachmentTooLarge) {
		t.Fatal("pixel bound", e)
	}
}

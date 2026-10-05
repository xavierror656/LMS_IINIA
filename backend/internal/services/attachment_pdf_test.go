package services

import (
	"aulaquest/internal/models"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// buildPDF returns a genuine PDF with a correct cross-reference table, so the
// structural analysis is exercised against a real file instead of a fake header.
func buildPDF(pages int, extra string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := []int{}
	add := func(body string) {
		offsets = append(offsets, b.Len())
		b.WriteString(body)
	}
	kids := make([]string, pages)
	for i := range kids {
		kids[i] = fmt.Sprintf("%d 0 R", 3+i)
	}
	add("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	add(fmt.Sprintf("2 0 obj\n<< /Type /Pages /Kids [%s] /Count %d >>\nendobj\n", strings.Join(kids, " "), pages))
	for i := 0; i < pages; i++ {
		add(fmt.Sprintf("%d 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>\nendobj\n", 3+i))
	}
	if extra != "" {
		add(extra)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xref)
	return b.Bytes()
}

func TestPDFStructureIsAnalyzed(t *testing.T) {
	valid := buildPDF(1, "")
	data, kind, e := NormalizeAttachment("tarea.pdf", valid)
	if e != nil {
		t.Fatalf("a valid PDF was rejected: %v", e)
	}
	if kind != "application/pdf" || !bytes.Equal(data, valid) {
		t.Fatalf("a PDF is stored unchanged as application/pdf, got %q", kind)
	}
	if n := pdfPages(valid); n != 1 {
		t.Fatalf("expected one page, counted %d", n)
	}
	if n := pdfPages(buildPDF(3, "")); n != 3 {
		t.Fatalf("expected three pages, counted %d", n)
	}
	// The page tree node must not be counted as a page.
	if n := pdfPages(buildPDF(1, "4 0 obj\n<< /Type /Pages /Kids [] /Count 0 >>\nendobj\n")); n != 1 {
		t.Fatalf("a page tree node was counted as a page: %d", n)
	}
	assertRealPDF(t, valid)
}

// assertRealPDF proves the fixture is a genuine file and not something shaped to
// please the analyzer: every cross-reference offset must point at its own object.
func assertRealPDF(t *testing.T, data []byte) {
	t.Helper()
	start := bytes.Index(data, []byte("xref\n0 "))
	if start < 0 {
		t.Fatal("the fixture has no cross-reference table")
	}
	lines := bytes.Split(data[start:], []byte("\n"))
	// lines[0] is the keyword, lines[1] the subsection header and lines[2] the free
	// entry of object zero; object one starts at lines[3].
	if len(lines) < 4 {
		t.Fatal("the fixture has no cross-reference entries")
	}
	number := 1
	for _, line := range lines[3:] {
		if len(line) < 18 {
			break
		}
		var offset int
		if _, e := fmt.Sscanf(string(line[:10]), "%d", &offset); e != nil {
			t.Fatalf("unreadable xref entry %q", line)
		}
		head := fmt.Sprintf("%d 0 obj", number)
		if !bytes.HasPrefix(data[offset:], []byte(head)) {
			t.Fatalf("offset %d does not point at object %d", offset, number)
		}
		number++
	}
	if number < 3 {
		t.Fatalf("expected at least two objects, read %d", number-1)
	}
}
func TestPDFRejectsDangerousOrBrokenFiles(t *testing.T) {
	cases := map[string][]byte{
		"header":         append([]byte("not a pdf at all"), buildPDF(1, "")...),
		"no trailer":     buildPDF(1, "")[:len(buildPDF(1, ""))-8],
		"no xref":        []byte("%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<< /Size 2 >>\n%%EOF\n"),
		"encrypted":      buildPDF(1, "4 0 obj\n<< /Encrypt 5 0 R >>\nendobj\n"),
		"javascript":     buildPDF(1, "4 0 obj\n<< /Type /Action /S /JavaScript /JS (app.alert(1)) >>\nendobj\n"),
		"embedded":       buildPDF(1, "4 0 obj\n<< /Type /Filespec /EmbeddedFile 5 0 R >>\nendobj\n"),
		"launch":         buildPDF(1, "4 0 obj\n<< /Type /Action /S /Launch /F (cmd.exe) >>\nendobj\n"),
		"too many pages": buildPDF(maxPDFPages+1, ""),
		"empty file":     {},
	}
	for name, data := range cases {
		if _, _, e := NormalizeAttachment("tarea.pdf", data); e == nil {
			t.Fatalf("%s: a broken or dangerous PDF was accepted", name)
		}
	}
}
func TestPDFPagesOverLimitIsTooLarge(t *testing.T) {
	_, _, e := NormalizeAttachment("tarea.pdf", buildPDF(maxPDFPages+1, ""))
	if !errors.Is(e, ErrAttachmentTooLarge) {
		t.Fatalf("a document with too many pages must be too large, got %v", e)
	}
}
func TestPDFNameNeedsADelimiter(t *testing.T) {
	// A declared name ends at a delimiter; a longer name is a different feature.
	if !hasPDFName([]byte("<< /JS >>"), []byte("/JS")) {
		t.Fatal("a declared /JS name must be found")
	}
	if hasPDFName([]byte("<< /JSDictionary >>"), []byte("/JS")) {
		t.Fatal("a longer name must not match")
	}
	if !hasPDFName([]byte("<< /Type/Action/S/JavaScript >>"), []byte("/JavaScript")) {
		t.Fatal("a name followed by a solidus must match")
	}
}
func TestOtherFormatsKeepTheirRules(t *testing.T) {
	if _, _, e := NormalizeAttachment("notas.txt", []byte("hola")); e != nil {
		t.Fatalf("plain text must keep working: %v", e)
	}
	// A PDF renamed as an image is rejected by the real image decoder.
	if _, _, e := NormalizeAttachment("dibujo.png", buildPDF(1, "")); e == nil {
		t.Fatal("a spoofed image extension was accepted")
	}
	if _, _, e := NormalizeAttachment("script.exe", []byte("MZ")); !errors.Is(e, models.ErrAcademicInput) {
		t.Fatalf("an unknown extension must be refused, got %v", e)
	}
}

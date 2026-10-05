package handlers

import (
	"aulaquest/internal/middleware"
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"mime"
	"strconv"
)

func (a API) gradebook(c *fiber.Ctx) error {
	id, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	p, e := page(c)
	if e != nil {
		return e
	}
	ap, e := strconv.Atoi(c.Query("activityPage", "1"))
	if e != nil || ap < 1 || ap > 10000 {
		return fiber.ErrBadRequest
	}
	out, e := a.academic().Gradebook(middleware.User(c).ID, id, p, ap)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}

// meGrades is the student's own view of the book: published categories and
// totals only, never drafts or hidden grades.
func (a API) meGrades(c *fiber.Ctx) error {
	course, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	out, e := a.academic().StudentGrades(middleware.User(c).ID, course)
	if e != nil {
		return academicError(e)
	}
	return c.JSON(out)
}

// gradebookCSV downloads the book with the same numbers the table shows. Only
// published grades travel: an unpublished one stays on the teacher's screen.
func (a API) gradebookCSV(c *fiber.Ctx) error {
	id, e := ID(c, "courseId")
	if e != nil {
		return e
	}
	book, overflow, e := a.academic().GradebookExport(middleware.User(c).ID, id)
	if e != nil {
		return academicError(e)
	}
	if overflow {
		return fiber.NewError(409, "Este curso supera el límite de descarga. Pide la exportación por partes o contacta con la administración.")
	}
	var buf bytes.Buffer
	// A spreadsheet reads a UTF-8 CSV with accents correctly only with the mark.
	buf.WriteString("\ufeff")
	writer := csv.NewWriter(&buf)
	header := []string{"Estudiante"}
	for _, activity := range book.Activities {
		header = append(header, csvSafe(activity.Title))
	}
	for _, category := range book.Categories {
		header = append(header, csvSafe("Total "+category.Name))
	}
	header = append(header, "Total del curso", "Peso publicado", "Promedio publicado", "Entregadas", "Calificadas", "Publicadas", "Pendientes de revisión", "Pendientes de publicación", "Sin entregar")
	if e = writer.Write(header); e != nil {
		return dbError(e)
	}
	for _, row := range book.Rows {
		line := []string{csvSafe(row.Alias)}
		for _, cell := range row.Cells {
			value := ""
			if cell.State == "published" && cell.Score != nil {
				value = strconv.Itoa(*cell.Score)
			}
			line = append(line, value)
		}
		totals := map[int64]*int64{}
		for _, total := range row.Summary.CategoryTotals {
			totals[total.CategoryID] = total.TotalHundredths
		}
		for _, category := range book.Categories {
			line = append(line, hundredths(totals[category.ID]))
		}
		line = append(line,
			hundredths(row.Summary.CourseTotalHundredths),
			strconv.FormatInt(row.Summary.PublishedWeight, 10),
			hundredths(row.Summary.WeightedAverageHundredths),
			strconv.FormatInt(row.Summary.Published+row.Summary.PendingReview+row.Summary.PendingPublication, 10),
			strconv.FormatInt(row.Summary.Published+row.Summary.PendingPublication, 10),
			strconv.FormatInt(row.Summary.Published, 10),
			strconv.FormatInt(row.Summary.PendingReview, 10),
			strconv.FormatInt(row.Summary.PendingPublication, 10),
			strconv.FormatInt(row.Summary.NotSubmitted, 10))
		if e = writer.Write(line); e != nil {
			return dbError(e)
		}
	}
	writer.Flush()
	if e = writer.Error(); e != nil {
		return dbError(e)
	}
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "libro-" + book.Course.Slug + ".csv"}))
	c.Set("Cache-Control", "private, no-store")
	c.Set("X-Content-Type-Options", "nosniff")
	return c.Send(buf.Bytes())
}

// csvSafe stops a spreadsheet from reading a name as a formula.
func csvSafe(value string) string {
	if value == "" {
		return value
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + value
	}
	return value
}

// hundredths renders an exact hundredths value as a decimal with two places.
func hundredths(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value/100, 10) + "." + fmt.Sprintf("%02d", *value%100)
}

package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
)

// The screen paginates; the export takes everything up to hard caps. Both go
// through the same query and the same assembly, so the file and the table always
// show the same numbers.
const (
	GradebookPageSize         = 20
	GradebookActivityPageSize = 10
)

// The export caps are variables so a test can exercise the refusal without
// creating five hundred students.
var (
	MaxExportStudents   int64 = 500
	MaxExportActivities int64 = 100
)

// PublishedAverage returns an exact hundredths value, rounding half upward.
// Missing grades are not zero. Scores are integers between 0 and 100.
func PublishedAverage(sum, count int64) *int64 {
	if count == 0 {
		return nil
	}
	value := (sum*100 + count/2) / count
	return &value
}

// CategoryAverage turns a student's raw sums inside one category into an exact
// hundredths value, rounding half upward. The course missing policy decides the
// denominator: `exclude` weighs only published grades, `zero` weighs every
// published activity and counts an absent grade as zero. publishedSum already
// carries only published scores in both cases.
func CategoryAverage(publishedSum, publishedWeight, totalWeight int64, policy string) *int64 {
	weight := publishedWeight
	if policy == "zero" {
		weight = totalWeight
	}
	if weight == 0 {
		return nil
	}
	value := (publishedSum*100 + weight/2) / weight
	return &value
}

// WeightedHundredths combines already-rounded category totals by category
// weight. Categories without a total do not participate; nothing becomes zero.
func WeightedHundredths(values, weights []int64) *int64 {
	var sum, weight int64
	for i := range values {
		sum += values[i] * weights[i]
		weight += weights[i]
	}
	if weight == 0 {
		return nil
	}
	value := (sum + weight/2) / weight
	return &value
}

func (s AcademicService) Gradebook(user, course int64, page, activityPage int) (models.Gradebook, error) {
	if page < 1 || page > 10000 || activityPage < 1 || activityPage > 10000 {
		return models.Gradebook{}, models.ErrAcademicInput
	}
	data, e := s.Repo.Gradebook(user, course, page, GradebookPageSize, activityPage, GradebookActivityPageSize)
	if e != nil {
		return models.Gradebook{}, e
	}
	return assembleGradebook(data, page, GradebookPageSize, activityPage, GradebookActivityPageSize), nil
}

// GradebookExport returns the whole course book for a download. It reports overflow
// instead of silently truncating: a partial file would look complete.
func (s AcademicService) GradebookExport(user, course int64) (models.Gradebook, bool, error) {
	data, e := s.Repo.Gradebook(user, course, 1, int(MaxExportStudents)+1, 1, int(MaxExportActivities)+1)
	if e != nil {
		return models.Gradebook{}, false, e
	}
	overflow := int64(len(data.Students)) > MaxExportStudents || int64(len(data.Activities)) > MaxExportActivities
	if overflow {
		// Trim before assembling so nothing downstream builds a giant response.
		if int64(len(data.Students)) > MaxExportStudents {
			data.Students = data.Students[:MaxExportStudents]
		}
		if int64(len(data.Activities)) > MaxExportActivities {
			data.Activities = data.Activities[:MaxExportActivities]
		}
	}
	return assembleGradebook(data, 1, int(MaxExportStudents), 1, int(MaxExportActivities)), overflow, nil
}

// assembleGradebook turns the query rows into the grid the book and the export share.
func assembleGradebook(data repositories.GradebookData, page, pageSize, activityPage, activityPageSize int) models.Gradebook {
	out := models.Gradebook{Course: data.Course, MissingPolicy: data.MissingPolicy, Categories: data.Categories, Activities: data.Activities, Rows: []models.GradebookRow{}, Page: page, PageSize: pageSize, TotalStudents: data.TotalStudents, ActivityPage: activityPage, ActivityPageSize: activityPageSize, TotalActivities: data.TotalActivities}
	entries := map[[2]int64]repositories.GradebookEntry{}
	for _, entry := range data.Entries {
		entries[[2]int64{entry.StudentID, entry.ActivityID}] = entry
	}
	categorySums := map[[2]int64]repositories.GradebookCategorySum{}
	for _, sum := range data.CategorySums {
		categorySums[[2]int64{sum.StudentID, sum.CategoryID}] = sum
	}
	for _, student := range data.Students {
		row := models.GradebookRow{StudentID: student.ID, Alias: student.Alias, Cells: []models.GradebookCell{}, Summary: models.GradebookSummary{
			TotalWeight: data.TotalWeight, PublishedWeight: student.PublishedWeight, WeightedAverageHundredths: PublishedAverage(student.WeightedSum, student.PublishedWeight),
			TotalActivities: data.TotalActivities, NotSubmitted: data.TotalActivities - student.Submitted,
			PendingReview: student.Submitted - student.Graded, PendingPublication: student.Graded - student.Published,
			Published: student.Published, AverageHundredths: PublishedAverage(student.PublishedSum, student.Published),
		}}
		var totals, weights []int64
		for _, category := range data.Categories {
			sum := categorySums[[2]int64{student.ID, category.ID}]
			total := CategoryAverage(sum.PublishedSum, sum.PublishedWeight, sum.TotalWeight, data.MissingPolicy)
			row.Summary.CategoryTotals = append(row.Summary.CategoryTotals, models.GradebookCategoryTotal{CategoryID: category.ID, TotalHundredths: total})
			if total != nil {
				totals = append(totals, *total)
				weights = append(weights, int64(category.Weight))
			}
		}
		if row.Summary.CategoryTotals == nil {
			row.Summary.CategoryTotals = []models.GradebookCategoryTotal{}
		}
		row.Summary.CourseTotalHundredths = WeightedHundredths(totals, weights)
		for _, activity := range data.Activities {
			cell := models.GradebookCell{ActivityID: activity.ActivityID, State: "not_submitted"}
			if entry, ok := entries[[2]int64{student.ID, activity.ActivityID}]; ok {
				cell.State = "submitted"
				if entry.QuizAttemptID != nil {
					cell.QuizAttemptID = entry.QuizAttemptID
				} else {
					cell.SubmissionID = &entry.SubmissionID
				}
				if entry.GradeStatus != nil {
					cell.State = "graded"
					cell.Score = entry.Score
					if *entry.GradeStatus == "published" {
						cell.State = "published"
					}
				}
			}
			row.Cells = append(row.Cells, cell)
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

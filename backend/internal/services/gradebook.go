package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
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

func (s AcademicService) Gradebook(user, course int64, page, activityPage int) (models.Gradebook, error) {
	if page < 1 || page > 10000 || activityPage < 1 || activityPage > 10000 {
		return models.Gradebook{}, models.ErrAcademicInput
	}
	data, e := s.Repo.Gradebook(user, course, page, activityPage)
	if e != nil {
		return models.Gradebook{}, e
	}
	out := models.Gradebook{Course: data.Course, Activities: data.Activities, Rows: []models.GradebookRow{}, Page: page, PageSize: 20, TotalStudents: data.TotalStudents, ActivityPage: activityPage, ActivityPageSize: 10, TotalActivities: data.TotalActivities}
	entries := map[[2]int64]repositories.GradebookEntry{}
	for _, entry := range data.Entries {
		entries[[2]int64{entry.StudentID, entry.ActivityID}] = entry
	}
	for _, student := range data.Students {
		row := models.GradebookRow{StudentID: student.ID, Alias: student.Alias, Cells: []models.GradebookCell{}, Summary: models.GradebookSummary{
			TotalActivities: data.TotalActivities, NotSubmitted: data.TotalActivities - student.Submitted,
			PendingReview: student.Submitted - student.Graded, PendingPublication: student.Graded - student.Published,
			Published: student.Published, AverageHundredths: PublishedAverage(student.PublishedSum, student.Published),
		}}
		for _, activity := range data.Activities {
			cell := models.GradebookCell{ActivityID: activity.ActivityID, State: "not_submitted"}
			if entry, ok := entries[[2]int64{student.ID, activity.ActivityID}]; ok {
				cell.State = "submitted"
				cell.SubmissionID = &entry.SubmissionID
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
	return out, nil
}

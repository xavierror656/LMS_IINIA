package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
)

// StudentGrades assembles one student's own categories and totals from the same
// aggregation the teacher book uses. Only published grades ever travel.
func (s AcademicService) StudentGrades(student, course int64) (models.StudentGrades, error) {
	data, e := s.Repo.StudentGrades(student, course)
	if e != nil {
		return models.StudentGrades{}, e
	}
	out := models.StudentGrades{Course: data.Course, MissingPolicy: data.MissingPolicy, Categories: []models.StudentGradeCategory{}}
	sums := map[int64]repositories.GradebookCategorySum{}
	for _, sum := range data.CategorySums {
		sums[sum.CategoryID] = sum
	}
	items := map[int64][]models.StudentGradeItem{}
	for _, item := range data.Items {
		items[item.CategoryID] = append(items[item.CategoryID], models.StudentGradeItem{Score: item.Score, LessonID: item.LessonID, Title: item.Title, Type: item.Type, Weight: item.Weight})
	}
	var totals, weights []int64
	for _, category := range data.Categories {
		sum := sums[category.ID]
		total := CategoryAverage(sum.PublishedSum, sum.PublishedWeight, sum.TotalWeight, data.MissingPolicy)
		content := items[category.ID]
		if content == nil {
			content = []models.StudentGradeItem{}
		}
		out.Categories = append(out.Categories, models.StudentGradeCategory{ID: category.ID, Name: category.Name, Weight: category.Weight, TotalHundredths: total, Items: content})
		if total != nil {
			totals = append(totals, *total)
			weights = append(weights, int64(category.Weight))
		}
	}
	out.CourseTotalHundredths = WeightedHundredths(totals, weights)
	return out, nil
}

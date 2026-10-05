package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"gorm.io/gorm"
	"strings"
)

// lockCourse serializes category mutations per course so positions and names
// cannot race between two teachers of the same course.
func lockCourse(tx *gorm.DB, course int64) error {
	var id int64
	if e := tx.Raw(`SELECT id FROM courses WHERE id=? FOR UPDATE`, course).Scan(&id).Error; e != nil {
		return e
	}
	if id == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// validateActivityCategory rejects a category that does not belong to the
// activity's course. A nil category is allowed: publishing resolves the default.
func validateActivityCategory(tx *gorm.DB, module int64, categoryID *int64) error {
	if categoryID == nil {
		return nil
	}
	var found int64
	if e := tx.Raw(`SELECT c.id FROM grade_categories c JOIN modules m ON m.course_id=c.course_id WHERE c.id=? AND m.id=?`, *categoryID, module).Scan(&found).Error; e != nil {
		return e
	}
	if found == 0 {
		return models.ErrAcademicInput
	}
	return nil
}

// ensureActivityCategory resolves the category a graded activity will carry when
// published: the explicit draft choice or the course's first category, creating
// General when the course still has none. Readings pass explicit nil and get nil.
func ensureActivityCategory(tx *gorm.DB, module int64, explicit *int64) (any, error) {
	if explicit != nil {
		return *explicit, nil
	}
	var course int64
	if e := tx.Raw(`SELECT course_id FROM modules WHERE id=?`, module).Scan(&course).Error; e != nil {
		return nil, e
	}
	var id int64
	if e := tx.Raw(`SELECT id FROM grade_categories WHERE course_id=? ORDER BY position,id LIMIT 1`, course).Scan(&id).Error; e != nil {
		return nil, e
	}
	if id == 0 {
		if e := tx.Raw(`INSERT INTO grade_categories(course_id,name,weight,position) VALUES (?,'General',1,1) RETURNING id`, course).Scan(&id).Error; e != nil {
			return nil, e
		}
	}
	return id, nil
}

func (s AcademicService) Categories(user, course int64) ([]models.GradeCategory, string, error) {
	out := []models.GradeCategory{}
	var policy string
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).StaffCourse(user, course); e != nil {
			return e
		}
		if e := tx.Raw(`SELECT missing_policy FROM courses WHERE id=?`, course).Scan(&policy).Error; e != nil {
			return e
		}
		return tx.Raw(`SELECT id,course_id,name,weight,position,version FROM grade_categories WHERE course_id=? ORDER BY position,id`, course).Scan(&out).Error
	})
	return out, policy, e
}

func (s AcademicService) CreateCategory(user, course int64, input models.GradeCategoryInput) (models.GradeCategory, error) {
	var out models.GradeCategory
	name, weight, e := input.Normalized()
	if e != nil {
		return out, e
	}
	e = s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).StaffCourse(user, course); e != nil {
			return e
		}
		if e := lockCourse(tx, course); e != nil {
			return e
		}
		var count int64
		if e := tx.Raw(`SELECT count(*) FROM grade_categories WHERE course_id=?`, course).Scan(&count).Error; e != nil {
			return e
		}
		if count >= models.MaxGradeCategories {
			return models.ErrCategoryLimit
		}
		var taken int64
		if e := tx.Raw(`SELECT count(*) FROM grade_categories WHERE course_id=? AND lower(btrim(name))=lower(?)`, course, name).Scan(&taken).Error; e != nil {
			return e
		}
		if taken > 0 {
			return models.ErrCategoryNameTaken
		}
		return tx.Raw(`INSERT INTO grade_categories(course_id,name,weight,position)
      SELECT ?,?,?,COALESCE(MAX(position),0)+1 FROM grade_categories WHERE course_id=?
      RETURNING id,course_id,name,weight,position,version`, course, name, weight, course).Scan(&out).Error
	})
	return out, e
}

func (s AcademicService) SaveCategory(user, course, id int64, version int, update models.GradeCategoryUpdate) (models.GradeCategory, error) {
	var out models.GradeCategory
	name := strings.TrimSpace(update.Name)
	if version < 1 || update.Validate() != nil {
		return out, models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).StaffCourse(user, course); e != nil {
			return e
		}
		if e := lockCourse(tx, course); e != nil {
			return e
		}
		var current models.GradeCategory
		if e := tx.Raw(`SELECT id,version FROM grade_categories WHERE id=? AND course_id=?`, id, course).Scan(&current).Error; e != nil {
			return e
		}
		if current.ID == 0 {
			return gorm.ErrRecordNotFound
		}
		if current.Version != version {
			return ErrAcademicConflict
		}
		var taken int64
		if e := tx.Raw(`SELECT count(*) FROM grade_categories WHERE course_id=? AND lower(btrim(name))=lower(?) AND id<>?`, course, name, id).Scan(&taken).Error; e != nil {
			return e
		}
		if taken > 0 {
			return models.ErrCategoryNameTaken
		}
		return tx.Raw(`UPDATE grade_categories SET name=?,weight=?,version=version+1 WHERE id=?
      RETURNING id,course_id,name,weight,position,version`, name, update.Weight, id).Scan(&out).Error
	})
	return out, e
}

func (s AcademicService) DeleteCategory(user, course, id int64) error {
	return s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).StaffCourse(user, course); e != nil {
			return e
		}
		if e := lockCourse(tx, course); e != nil {
			return e
		}
		var exists int64
		if e := tx.Raw(`SELECT count(*) FROM grade_categories WHERE id=? AND course_id=?`, id, course).Scan(&exists).Error; e != nil {
			return e
		}
		if exists == 0 {
			return gorm.ErrRecordNotFound
		}
		var used int64
		if e := tx.Raw(`SELECT (SELECT count(*) FROM lessons WHERE grade_category_id=?)
      +(SELECT count(*) FROM authored_activities WHERE grade_category_id=?)
      +(SELECT count(*) FROM activity_publications WHERE grade_category_id=?)`, id, id, id).Scan(&used).Error; e != nil {
			return e
		}
		if used > 0 {
			return models.ErrCategoryHasActivities
		}
		return tx.Exec(`DELETE FROM grade_categories WHERE id=?`, id).Error
	})
}

func (s AcademicService) ReorderCategories(user, course int64, ids []int64) ([]models.GradeCategory, error) {
	out := []models.GradeCategory{}
	if len(ids) == 0 || len(ids) > models.MaxGradeCategories {
		return out, models.ErrAcademicInput
	}
	seen := map[int64]bool{}
	for _, id := range ids {
		if id < 1 || seen[id] {
			return out, models.ErrAcademicInput
		}
		seen[id] = true
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).StaffCourse(user, course); e != nil {
			return e
		}
		if e := lockCourse(tx, course); e != nil {
			return e
		}
		var matches int64
		if e := tx.Raw(`SELECT count(*) FROM grade_categories WHERE course_id=? AND id IN ?`, course, ids).Scan(&matches).Error; e != nil {
			return e
		}
		var total int64
		if e := tx.Raw(`SELECT count(*) FROM grade_categories WHERE course_id=?`, course).Scan(&total).Error; e != nil {
			return e
		}
		// The list must cover exactly the course categories: no missing, extra or
		// foreign identifiers.
		if matches != int64(len(ids)) || total != int64(len(ids)) {
			return models.ErrAcademicInput
		}
		for i, id := range ids {
			if e := tx.Exec(`UPDATE grade_categories SET position=? WHERE id=? AND course_id=?`, i+1, id, course).Error; e != nil {
				return e
			}
		}
		return tx.Raw(`SELECT id,course_id,name,weight,position,version FROM grade_categories WHERE course_id=? ORDER BY position,id`, course).Scan(&out).Error
	})
	return out, e
}

func (s AcademicService) SaveMissingPolicy(user, course int64, policy string) (string, error) {
	if !models.ValidMissingPolicy(policy) {
		return "", models.ErrAcademicInput
	}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).StaffCourse(user, course); e != nil {
			return e
		}
		if e := lockCourse(tx, course); e != nil {
			return e
		}
		return tx.Exec(`UPDATE courses SET missing_policy=? WHERE id=?`, policy, course).Error
	})
	return policy, e
}

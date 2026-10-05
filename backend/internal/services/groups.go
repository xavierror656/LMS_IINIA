package services

import (
	"aulaquest/internal/models"
	"aulaquest/internal/repositories"
	"errors"
	"gorm.io/gorm"
	"strings"
)

// ErrGroupMemberElsewhere means the student already belongs to another group of
// the same course: a student belongs to at most one group per course.
var ErrGroupMemberElsewhere = errors.New("student already belongs to another group in this course")

// ErrGroupNameTaken means the course already has a group with that name.
var ErrGroupNameTaken = errors.New("group name already used in this course")

// ErrGroupHasDeliveries means the group already owns submissions, so deleting it
// would orphan academic work.
var ErrGroupHasDeliveries = errors.New("group already owns deliveries")

func (s AcademicService) Groups(user, course int64, page int) ([]models.Group, error) {
	out := []models.Group{}
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).StaffCourse(user, course); e != nil {
			return e
		}
		return tx.Raw(`SELECT g.id,g.course_id,g.name,g.version,count(gm.user_id) members FROM course_groups g
  LEFT JOIN group_members gm ON gm.group_id=g.id WHERE g.course_id=? GROUP BY g.id ORDER BY g.id LIMIT 20 OFFSET ?`, course, (page-1)*20).Scan(&out).Error
	})
	return out, e
}

func (s AcademicService) CreateGroup(user, course int64, input models.GroupInput) (models.Group, error) {
	var out models.Group
	if input.Validate() != nil {
		return out, models.ErrAcademicInput
	}
	name := strings.TrimSpace(input.Name)
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).StaffCourse(user, course); e != nil {
			return e
		}
		// The unique index decides; an empty result means the name is taken.
		out.ID = 0
		return tx.Raw(`INSERT INTO course_groups(course_id,name,created_by) VALUES (?,?,?)
  ON CONFLICT (course_id,name) DO NOTHING RETURNING id,course_id,name,version,0 AS members`, course, name, user).Scan(&out).Error
	})
	if e == nil && out.ID == 0 {
		return out, ErrGroupNameTaken
	}
	return out, e
}

func (s AcademicService) SaveGroup(user, course, id int64, version int, input models.GroupInput) (models.Group, error) {
	var out models.Group
	if version < 1 || input.Validate() != nil {
		return out, models.ErrAcademicInput
	}
	name := strings.TrimSpace(input.Name)
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).StaffCourse(user, course); e != nil {
			return e
		}
		out.ID = 0
		if e := tx.Raw(`UPDATE course_groups SET name=?,version=version+1 WHERE id=? AND course_id=? AND version=?
  AND NOT EXISTS(SELECT 1 FROM course_groups other WHERE other.course_id=course_groups.course_id AND other.name=? AND other.id<>course_groups.id)
  RETURNING id,course_id,name,version,(SELECT count(*) FROM group_members WHERE group_id=course_groups.id) AS members`, name, id, course, version, name).Scan(&out).Error; e != nil {
			return e
		}
		if out.ID != 0 {
			return nil
		}
		// Nothing changed: tell a missing group from a stale revision or a taken name.
		var current struct {
			Version int
		}
		if e := tx.Raw(`SELECT version FROM course_groups WHERE id=? AND course_id=?`, id, course).Scan(&current).Error; e != nil {
			return e
		}
		if current.Version == 0 {
			return gorm.ErrRecordNotFound
		}
		var taken bool
		if e := tx.Raw(`SELECT EXISTS(SELECT 1 FROM course_groups other WHERE other.course_id=? AND other.name=? AND other.id<>?)`, course, name, id).Scan(&taken).Error; e != nil {
			return e
		}
		if taken {
			return ErrGroupNameTaken
		}
		return ErrAcademicConflict
	})
	return out, e
}

func (s AcademicService) DeleteGroup(user, course, id int64) error {
	return s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).StaffCourse(user, course); e != nil {
			return e
		}
		// A group that already owns deliveries is part of the academic record: it is
		// never deleted, only emptied, so no work or grade is lost.
		var deliveries int64
		if e := tx.Raw(`SELECT count(*) FROM submissions WHERE group_id=?`, id).Scan(&deliveries).Error; e != nil {
			return e
		}
		if deliveries > 0 {
			return ErrGroupHasDeliveries
		}
		// Membership goes with the group; no submission, grade or progress is touched.
		if e := tx.Exec(`DELETE FROM group_members WHERE group_id=? AND course_id=?`, id, course).Error; e != nil {
			return e
		}
		var deleted int64
		if e := tx.Raw(`DELETE FROM course_groups WHERE id=? AND course_id=? RETURNING id`, id, course).Scan(&deleted).Error; e != nil {
			return e
		}
		if deleted == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (s AcademicService) AddGroupMember(user, course, group, student int64) (models.GroupRosterEntry, error) {
	var out models.GroupRosterEntry
	e := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).StaffCourse(user, course); e != nil {
			return e
		}
		var name string
		if e := tx.Raw(`SELECT name FROM course_groups WHERE id=? AND course_id=? FOR UPDATE`, group, course).Scan(&name).Error; e != nil {
			return e
		}
		if name == "" {
			return gorm.ErrRecordNotFound
		}
		// Same base as the extension roster: enrolled in the course and linked here.
		var alias string
		if e := tx.Raw(`SELECT u.alias FROM users u JOIN enrollments e ON e.user_id=u.id JOIN teacher_students ts ON ts.student_id=u.id
  WHERE u.id=? AND u.role='student' AND e.course_id=? AND ts.teacher_id=? FOR SHARE OF e,ts`, student, course, user).Scan(&alias).Error; e != nil {
			return e
		}
		if alias == "" {
			return gorm.ErrRecordNotFound
		}
		var current int64
		if e := tx.Raw(`SELECT group_id FROM group_members WHERE course_id=? AND user_id=?`, course, student).Scan(&current).Error; e != nil {
			return e
		}
		if current == group {
			// Repeating the same membership changes nothing.
			out = models.GroupRosterEntry{StudentID: student, Alias: alias, GroupID: group}
			return nil
		}
		if current != 0 {
			return ErrGroupMemberElsewhere
		}
		if e := tx.Exec(`INSERT INTO group_members(group_id,course_id,user_id) VALUES (?,?,?)`, group, course, student).Error; e != nil {
			return e
		}
		out = models.GroupRosterEntry{StudentID: student, Alias: alias, GroupID: group}
		return nil
	})
	return out, e
}

func (s AcademicService) RemoveGroupMember(user, course, group, student int64) error {
	return s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		if _, e := (repositories.Repository{DB: tx}).StaffCourse(user, course); e != nil {
			return e
		}
		var removed int64
		if e := tx.Raw(`DELETE FROM group_members WHERE group_id=? AND course_id=? AND user_id=? RETURNING user_id`, group, course, student).Scan(&removed).Error; e != nil {
			return e
		}
		if removed == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

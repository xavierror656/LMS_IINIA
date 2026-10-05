package repositories

import (
	"aulaquest/internal/models"
	"database/sql"
	"gorm.io/gorm"
)

// Aggregates include every published assignment in the course, not only the
// current ten columns. Draft responses do not participate in these queries.
type GradebookStudent struct {
	PublishedWeight int64
	WeightedSum     int64
	ID              int64
	Alias           string
	Submitted       int64
	Graded          int64
	Published       int64
	PublishedSum    int64
}
type GradebookEntry struct {
	QuizAttemptID *int64
	StudentID     int64
	ActivityID    int64
	SubmissionID  int64
	Score         *int
	GradeStatus   *string
}
type GradebookData struct {
	TotalWeight     int64
	Course          models.Course
	Activities      []models.GradebookActivity
	Students        []GradebookStudent
	Entries         []GradebookEntry
	TotalStudents   int64
	TotalActivities int64
}

// Gradebook builds one window of the book. The screen asks for a page of students
// and a page of activities; the export asks for everything up to its own caps, with
// the same queries, so both can never disagree.
func (r Repository) Gradebook(user, course int64, studentPage, studentSize, activityPage, activitySize int) (GradebookData, error) {
	out := GradebookData{Activities: []models.GradebookActivity{}, Students: []GradebookStudent{}, Entries: []GradebookEntry{}}
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var e error
		out.Course, e = (Repository{DB: tx}).StaffCourse(user, course)
		if e != nil {
			return e
		}
		if e = tx.Raw(`SELECT count(*) FROM users u JOIN enrollments e ON e.user_id=u.id
    JOIN teacher_students ts ON ts.student_id=u.id
    WHERE u.role='student' AND e.course_id=? AND ts.teacher_id=?`, course, user).Scan(&out.TotalStudents).Error; e != nil {
			return e
		}
		var totals struct {
			TotalActivities int64
			TotalWeight     int64
		}
		if e = tx.Raw(`SELECT count(*) total_activities,COALESCE(sum(l.grade_weight),0) total_weight FROM authored_activities a JOIN lessons l ON l.id=a.lesson_id
    JOIN modules m ON m.id=l.module_id WHERE m.course_id=? AND l.type IN ('assignment','quiz')`, course).Scan(&totals).Error; e != nil {
			return e
		}
		out.TotalActivities, out.TotalWeight = totals.TotalActivities, totals.TotalWeight
		if e = tx.Raw(`SELECT a.id activity_id,l.id lesson_id,l.title,l.type,l.grade_weight weight FROM authored_activities a
    JOIN lessons l ON l.id=a.lesson_id JOIN modules m ON m.id=l.module_id
    WHERE m.course_id=? AND l.type IN ('assignment','quiz') ORDER BY m.position,l.position,l.id LIMIT ? OFFSET ?`, course, activitySize, (activityPage-1)*activitySize).Scan(&out.Activities).Error; e != nil {
			return e
		}
		if e = tx.Raw(`WITH roster AS (
    SELECT u.id,u.alias FROM users u JOIN enrollments e ON e.user_id=u.id
    JOIN teacher_students ts ON ts.student_id=u.id
    WHERE u.role='student' AND e.course_id=? AND ts.teacher_id=? ORDER BY u.id LIMIT ? OFFSET ?
  ), assignments AS (
    SELECT l.id,l.grade_weight FROM authored_activities a JOIN lessons l ON l.id=a.lesson_id
    JOIN modules m ON m.id=l.module_id WHERE m.course_id=? AND l.type IN ('assignment','quiz')
  ), scoped AS MATERIALIZED (
    SELECT e.user_id,e.lesson_id,e.score,e.status FROM gradebook_entries e WHERE e.lesson_id IN (SELECT id FROM assignments)
  ) SELECT r.id,r.alias,count(e.lesson_id) submitted,count(e.score) graded,
    count(e.lesson_id) FILTER(WHERE e.status='published') published,
    COALESCE(sum(e.score) FILTER(WHERE e.status='published'),0) published_sum,
    COALESCE(sum(e.score::bigint*aw.grade_weight) FILTER(WHERE e.status='published'),0) weighted_sum,
    COALESCE(sum(aw.grade_weight) FILTER(WHERE e.status='published'),0) published_weight
    FROM roster r LEFT JOIN scoped e ON e.user_id=r.id
    LEFT JOIN assignments aw ON aw.id=e.lesson_id
    GROUP BY r.id,r.alias ORDER BY r.id`, course, user, studentSize, (studentPage-1)*studentSize, course).Scan(&out.Students).Error; e != nil {
			return e
		}
		students := []int64{}
		lessons := []int64{}
		for _, s := range out.Students {
			students = append(students, s.ID)
		}
		for _, a := range out.Activities {
			lessons = append(lessons, a.LessonID)
		}
		if len(students) > 0 && len(lessons) > 0 {
			// One batch for the visible grid; no query per student or activity.
			return tx.Raw(`SELECT e.user_id student_id,a.id activity_id,COALESCE(e.submission_id,0) submission_id,e.quiz_attempt_id,e.score,e.status grade_status
     FROM gradebook_entries e JOIN authored_activities a ON a.lesson_id=e.lesson_id
     WHERE e.user_id IN ? AND e.lesson_id IN ?`, students, lessons).Scan(&out.Entries).Error
		}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	return out, err
}

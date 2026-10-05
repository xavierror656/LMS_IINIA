package models

// GroupInput is the editable part of a course group.
type GroupInput struct {
	Name string `json:"name"`
}

func (g GroupInput) Validate() error {
	if !ValidText(g.Name, 1, 100) {
		return ErrAcademicInput
	}
	return nil
}

// Group is a course group. Membership is managed through the member routes.
type Group struct {
	ID       int64  `json:"id"`
	CourseID int64  `json:"courseId"`
	Name     string `json:"name"`
	Version  int    `json:"version"`
	Members  int    `json:"members"`
}

// GroupRosterEntry pairs a linked enrolled student with the group they belong to,
// zero when they have none. It keeps the group page to a single request.
type GroupRosterEntry struct {
	StudentID int64  `json:"studentId"`
	Alias     string `json:"alias"`
	GroupID   int64  `json:"groupId"`
}

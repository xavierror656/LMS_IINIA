package models

type GradebookActivity struct {
	Weight     int    `json:"weight"`
	ActivityID int64  `json:"activityId"`
	LessonID   int64  `json:"lessonId"`
	Title      string `json:"title"`
}
type GradebookCell struct {
	ActivityID   int64  `json:"activityId"`
	State        string `json:"state"`
	Score        *int   `json:"score"`
	SubmissionID *int64 `json:"submissionId"`
}
type GradebookSummary struct {
	WeightedAverageHundredths *int64 `json:"weightedAverageHundredths"`
	PublishedWeight           int64  `json:"publishedWeight"`
	TotalWeight               int64  `json:"totalWeight"`
	TotalActivities           int64  `json:"totalActivities"`
	NotSubmitted              int64  `json:"notSubmitted"`
	PendingReview             int64  `json:"pendingReview"`
	PendingPublication        int64  `json:"pendingPublication"`
	Published                 int64  `json:"published"`
	AverageHundredths         *int64 `json:"averageHundredths"`
}
type GradebookRow struct {
	StudentID int64            `json:"studentId"`
	Alias     string           `json:"alias"`
	Cells     []GradebookCell  `json:"cells"`
	Summary   GradebookSummary `json:"summary"`
}
type Gradebook struct {
	Course           Course              `json:"course"`
	Activities       []GradebookActivity `json:"activities"`
	Rows             []GradebookRow      `json:"rows"`
	Page             int                 `json:"page"`
	PageSize         int                 `json:"pageSize"`
	TotalStudents    int64               `json:"totalStudents"`
	ActivityPage     int                 `json:"activityPage"`
	ActivityPageSize int                 `json:"activityPageSize"`
	TotalActivities  int64               `json:"totalActivities"`
}

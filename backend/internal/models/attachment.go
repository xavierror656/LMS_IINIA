package models

type Attachment struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Size        int    `json:"size"`
	// UploadedBy is the student charged for the file. In a group delivery it is
	// whoever uploaded it, not the author of the submission.
	UploadedBy int64 `json:"uploadedBy"`
}

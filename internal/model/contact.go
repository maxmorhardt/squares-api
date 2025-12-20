package model

type ContactRequest struct {
	Name    string `json:"name" binding:"required,min=1,max=100" example:"John Doe" description:"Name of the person contacting"`
	Email   string `json:"email" binding:"required,email,max=255" example:"john@example.com" description:"Email address for response"`
	Subject string `json:"subject" binding:"required,min=1,max=200" example:"Question about features" description:"Subject of the contact message"`
	Message string `json:"message" binding:"required,min=1,max=2000" example:"I have a question..." description:"Message content"`
}

type UpdateContactSubmissionRequest struct {
	Status   *string `json:"status" binding:"omitempty,oneof=pending responded resolved" example:"responded" description:"Status of the contact submission"`
	Response *string `json:"response" binding:"omitempty,min=1,max=2000" example:"Thank you for contacting us..." description:"Admin response to the contact submission"`
}

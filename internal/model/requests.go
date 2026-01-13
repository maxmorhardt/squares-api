package model

type CreateContestRequest struct {
	Owner    string `json:"owner" binding:"required" description:"Owner's username" validate:"required"`
	Name     string `json:"name" binding:"required" example:"My Contest" validate:"required,max=20,min=1"`
	HomeTeam string `json:"homeTeam,omitempty" example:"Home Team" validate:"max=20"`
	AwayTeam string `json:"awayTeam,omitempty" example:"Away Team" validate:"max=20"`
}

type UpdateSquareRequest struct {
	Value string `json:"value" binding:"required,min=1,max=3,uppercase,alphanum" example:"MRM" validate:"required,max=3,min=1" description:"Square value (required)"`
	Owner string `json:"owner" binding:"required" example:"username" validate:"required" description:"Square owner (required)"`
}

type ClearSquareRequest struct{}

type UpdateContestRequest struct {
	Name     *string `json:"name,omitempty" example:"Updated Contest Name" validate:"omitempty,max=20,min=1"`
	HomeTeam *string `json:"homeTeam,omitempty" example:"Updated Home Team" validate:"omitempty,max=20"`
	AwayTeam *string `json:"awayTeam,omitempty" example:"Updated Away Team" validate:"omitempty,max=20"`
}

type QuarterResultRequest struct {
	HomeTeamScore int `json:"homeTeamScore" binding:"min=0" example:"14" description:"Home team score"`
	AwayTeamScore int `json:"awayTeamScore" binding:"min=0" example:"7" description:"Away team score"`
}

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

package model

type CreateContestRequest struct {
	Owner    string `json:"owner" binding:"required,max=255"`
	Name     string `json:"name" binding:"required,max=20,min=1"`
	HomeTeam string `json:"homeTeam,omitempty" binding:"max=20"`
	AwayTeam string `json:"awayTeam,omitempty" binding:"max=20"`
}

type UpdateSquareRequest struct {
	Value string `json:"value" binding:"required,min=1,max=3,uppercase,alphanum"`
	Owner string `json:"owner" binding:"required"`
}

type ClearSquareRequest struct{}

type UpdateContestRequest struct {
	HomeTeam *string `json:"homeTeam,omitempty" binding:"omitempty,max=20"`
	AwayTeam *string `json:"awayTeam,omitempty" binding:"omitempty,max=20"`
}

type QuarterResultRequest struct {
	HomeTeamScore int `json:"homeTeamScore" binding:"required,min=0,max=9999"`
	AwayTeamScore int `json:"awayTeamScore" binding:"required,min=0,max=9999"`
}

type ContactRequest struct {
	Name    string `json:"name" binding:"required,min=1,max=100"`
	Email   string `json:"email" binding:"required,email,max=255"`
	Subject string `json:"subject" binding:"required,min=1,max=200"`
	Message string `json:"message" binding:"required,min=1,max=2000"`
}

type UpdateContactSubmissionRequest struct {
	Status   *string `json:"status" binding:"omitempty,oneof=pending responded resolved"`
	Response *string `json:"response" binding:"omitempty,min=1,max=2000"`
}

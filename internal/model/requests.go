package model

type CreateContestRequest struct {
	Owner    string `json:"owner" binding:"required,max=255,safestring"`
	Name     string `json:"name" binding:"required,max=20,min=1,safestring"`
	HomeTeam string `json:"homeTeam,omitempty" binding:"max=20,safestring"`
	AwayTeam string `json:"awayTeam,omitempty" binding:"max=20,safestring"`
}

type UpdateSquareRequest struct {
	Value string `json:"value" binding:"required,min=1,max=3,uppercase,alphanum,safestring"`
	Owner string `json:"owner" binding:"required,safestring"`
}

type ClearSquareRequest struct{}

type UpdateContestRequest struct {
	HomeTeam *string `json:"homeTeam,omitempty" binding:"omitempty,max=20,safestring"`
	AwayTeam *string `json:"awayTeam,omitempty" binding:"omitempty,max=20,safestring"`
}

type QuarterResultRequest struct {
	HomeTeamScore int `json:"homeTeamScore" binding:"required,min=0,max=9999"`
	AwayTeamScore int `json:"awayTeamScore" binding:"required,min=0,max=9999"`
}

type ContactRequest struct {
	Name           string `json:"name" binding:"required,min=1,max=100,safestring"`
	Email          string `json:"email" binding:"required,email,max=255,safestring"`
	Subject        string `json:"subject" binding:"required,min=1,max=200,safestring"`
	Message        string `json:"message" binding:"required,min=1,max=2000,safestring"`
	TurnstileToken string `json:"turnstileToken" binding:"required,min=1,max=255,safestring"`
}

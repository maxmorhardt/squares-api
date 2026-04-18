package model

type CreateContestRequest struct {
	Owner      string `json:"owner" binding:"required,max=255,safestring"`
	Name       string `json:"name" binding:"required,max=20,min=1,safestring"`
	HomeTeam   string `json:"homeTeam,omitempty" binding:"max=20,safestring"`
	AwayTeam   string `json:"awayTeam,omitempty" binding:"max=20,safestring"`
	Visibility string `json:"visibility,omitempty" binding:"omitempty,oneof=private public"`
}

type UpdateSquareRequest struct {
	Value string `json:"value" binding:"required,min=1,max=3,uppercase,alphanum,safestring"`
	Owner string `json:"owner" binding:"required,safestring"`
}

type ClearSquareRequest struct{}

type UpdateContestRequest struct {
	HomeTeam   *string `json:"homeTeam,omitempty" binding:"omitempty,max=20,safestring"`
	AwayTeam   *string `json:"awayTeam,omitempty" binding:"omitempty,max=20,safestring"`
	Visibility *string `json:"visibility,omitempty" binding:"omitempty,oneof=private public"`
}

type QuarterResultRequest struct {
	HomeTeamScore int `json:"homeTeamScore" binding:"min=0,max=9999"`
	AwayTeamScore int `json:"awayTeamScore" binding:"min=0,max=9999"`
}

type CreateInviteRequest struct {
	MaxSquares int    `json:"maxSquares" binding:"required,min=1,max=100"`
	Role       string `json:"role" binding:"required,oneof=participant viewer"`
	MaxUses    int    `json:"maxUses,omitempty" binding:"min=0"`
	ExpiresIn  int    `json:"expiresIn,omitempty" binding:"min=0"` // minutes, 0 = no expiry
}

type UpdateParticipantRequest struct {
	Role       *string `json:"role,omitempty" binding:"omitempty,oneof=participant viewer"`
	MaxSquares *int    `json:"maxSquares,omitempty" binding:"omitempty,min=0,max=100"`
}

type ContactRequest struct {
	Name           string `json:"name" binding:"required,min=1,max=100,safestring"`
	Email          string `json:"email" binding:"required,email,max=255,safestring"`
	Subject        string `json:"subject" binding:"required,min=1,max=200,safestring"`
	Message        string `json:"message" binding:"required,min=1,max=2000,safestring"`
	TurnstileToken string `json:"turnstileToken" binding:"required,min=1,max=2000,safestring"`
}

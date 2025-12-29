package model

type HealthResponse struct {
	Status string `json:"status" example:"UP"`
}

type PaginatedContestResponse struct {
	Contests    []Contest `json:"contests"`
	Page        int       `json:"page"`
	Limit       int       `json:"limit"`
	Total       int64     `json:"total"`
	TotalPages  int       `json:"totalPages"`
	HasNext     bool      `json:"hasNext"`
	HasPrevious bool      `json:"hasPrevious"`
}

type GenerateInviteLinkResponse struct {
	Token     string `json:"token" example:"eyJhbGc..." description:"Invite token to append to contest URL"`
	ExpiresAt int64  `json:"expiresAt,omitempty" description:"Unix timestamp when token expires (if set)"`
}
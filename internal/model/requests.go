package model

type CreateContestRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateSquareRequest struct {
	Value string `json:"value" binding:"required"`
}

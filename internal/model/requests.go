package model

type CreateGridRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateGridCellRequest struct {
	Value string `json:"value" binding:"required"`
}
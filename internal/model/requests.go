package model

type CreateGridRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateGridCellRequest struct {
	Row   int    `json:"row" binding:"required"`
	Col   int    `json:"col" binding:"required"`
	Value string `json:"value" binding:"required"`
}

package model

import "github.com/google/uuid"

type CreateGridRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateGridCellRequest struct {
	Value string `json:"value" binding:"required"`
}

type SelectGridCellRequest struct {
	CellID uuid.UUID
}
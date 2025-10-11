package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	CellUpdateType       string = "cell_update"
	KeepAliveType        string = "keepalive"
	ConnectedType        string = "connected"
	ClosedConnectionType string = "connection_closed"
	GridChannelPrefix    string = "grid"
)

type GridChannelResponse struct {
	Type      string    `json:"type"`
	GridID    uuid.UUID `json:"gridId"`
	CellID    uuid.UUID `json:"cellId"`
	Value     string    `json:"value"`
	UpdatedBy string    `json:"updatedBy"`
	Timestamp time.Time `json:"timestamp"`
}

func NewKeepAliveMessage(gridId uuid.UUID) *GridChannelResponse {
	return &GridChannelResponse{
		Type:      KeepAliveType,
		GridID:    gridId,
		CellID:    uuid.Nil,
		Value:     "keepalive",
		UpdatedBy: "system",
		Timestamp: time.Now(),
	}
}

func NewConnectedMessage(gridId uuid.UUID, username string) *GridChannelResponse {
	return &GridChannelResponse{
		Type:      ConnectedType,
		GridID:    gridId,
		CellID:    uuid.Nil,
		Value:     "connected",
		UpdatedBy: "system",
		Timestamp: time.Now(),
	}
}

func NewClosedConnectionMessage(gridId uuid.UUID, username string) *GridChannelResponse {
	return &GridChannelResponse{
		Type:      ClosedConnectionType,
		GridID:    gridId,
		CellID:    uuid.Nil,
		Value:     "disconnected",
		UpdatedBy: "system",
		Timestamp: time.Now(),
	}
}

func NewCellUpdateMessage(gridId, cellId uuid.UUID, value, updatedBy string) *GridChannelResponse {
	return &GridChannelResponse{
		Type:      CellUpdateType,
		GridID:    gridId,
		CellID:    cellId,
		Value:     value,
		UpdatedBy: updatedBy,
		Timestamp: time.Now(),
	}
}

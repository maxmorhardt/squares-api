package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	SquareUpdateType     string = "square_update"
	ContestUpdateType    string = "contest_update"
	KeepAliveType        string = "keepalive"
	ConnectedType        string = "connected"
	ClosedConnectionType string = "connection_closed"
	ContestChannelPrefix string = "contest"
)

type ContestChannelResponse struct {
	Type      string    `json:"type"`
	ContestID uuid.UUID `json:"contestId"`
	SquareID  uuid.UUID `json:"squareId,omitempty"`
	Value     string    `json:"value,omitempty"`
	XLabels   []int8    `json:"xLabels,omitempty"`
	YLabels   []int8    `json:"yLabels,omitempty"`
	UpdatedBy string    `json:"updatedBy"`
	Timestamp time.Time `json:"timestamp"`
}

func NewKeepAliveMessage(contestId uuid.UUID) *ContestChannelResponse {
	return &ContestChannelResponse{
		Type:      KeepAliveType,
		ContestID: contestId,
		SquareID:  uuid.Nil,
		Value:     "keepalive",
		UpdatedBy: "system",
		Timestamp: time.Now(),
	}
}

func NewConnectedMessage(contestId uuid.UUID, username string) *ContestChannelResponse {
	return &ContestChannelResponse{
		Type:      ConnectedType,
		ContestID: contestId,
		SquareID:  uuid.Nil,
		Value:     "connected",
		UpdatedBy: "system",
		Timestamp: time.Now(),
	}
}

func NewClosedConnectionMessage(contestId uuid.UUID, username string) *ContestChannelResponse {
	return &ContestChannelResponse{
		Type:      ClosedConnectionType,
		ContestID: contestId,
		SquareID:  uuid.Nil,
		Value:     "disconnected",
		UpdatedBy: "system",
		Timestamp: time.Now(),
	}
}

func NewSquareUpdateMessage(contestId, squareId uuid.UUID, value, updatedBy string) *ContestChannelResponse {
	return &ContestChannelResponse{
		Type:      SquareUpdateType,
		ContestID: contestId,
		SquareID:  squareId,
		Value:     value,
		UpdatedBy: updatedBy,
		Timestamp: time.Now(),
	}
}

func NewContestUpdateMessage(contestId uuid.UUID, xLabels, yLabels []int8, updatedBy string) *ContestChannelResponse {
	return &ContestChannelResponse{
		Type:      ContestUpdateType,
		ContestID: contestId,
		SquareID:  uuid.Nil,
		XLabels:   xLabels,
		YLabels:   yLabels,
		UpdatedBy: updatedBy,
		Timestamp: time.Now(),
	}
}

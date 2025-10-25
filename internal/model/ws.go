package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	SquareUpdateType     string = "square_update"
	ContestUpdateType    string = "contest_update"
	ConnectedType        string = "connected"
	DisconnectType       string = "disconnected"
	ContestChannelPrefix string = "contest"
)

type WSUpdate struct {
	Type         string           `json:"type"`
	ContestID    uuid.UUID        `json:"contestId"`
	ConnectionID uuid.UUID        `json:"connectionId,omitempty"`
	UpdatedBy    string           `json:"updatedBy"`
	Timestamp    time.Time        `json:"timestamp"`
	Square       *SquareWSUpdate  `json:"square,omitempty"`
	Contest      *ContestWSUpdate `json:"contest,omitempty"`
}

type SquareWSUpdate struct {
	SquareID uuid.UUID `json:"squareId"`
	Value    string    `json:"value"`
}

type ContestWSUpdate struct {
	HomeTeam string        `json:"homeTeam,omitempty"`
	AwayTeam string        `json:"awayTeam,omitempty"`
	XLabels  []int8        `json:"xLabels,omitempty"`
	YLabels  []int8        `json:"yLabels,omitempty"`
	Status   ContestStatus `json:"status,omitempty"`
}

func NewConnectedMessage(contestId uuid.UUID, connectionID uuid.UUID) *WSUpdate {
	return &WSUpdate{
		Type:         ConnectedType,
		ContestID:    contestId,
		ConnectionID: connectionID,
		UpdatedBy:    "system",
		Timestamp:    time.Now(),
	}
}

func NewDisconnectedMessage(contestId uuid.UUID, connectionID uuid.UUID) *WSUpdate {
	return &WSUpdate{
		Type:         DisconnectType,
		ContestID:    contestId,
		ConnectionID: connectionID,
		UpdatedBy:    "system",
		Timestamp:    time.Now(),
	}
}

func NewSquareUpdateMessage(contestId uuid.UUID, updatedBy string, squareUpdate *SquareWSUpdate) *WSUpdate {
	return &WSUpdate{
		Type:      SquareUpdateType,
		ContestID: contestId,
		UpdatedBy: updatedBy,
		Timestamp: time.Now(),
		Square:    squareUpdate,
	}
}

func NewContestUpdateMessage(contestId uuid.UUID, updatedBy string, contestUpdate *ContestWSUpdate) *WSUpdate {
	return &WSUpdate{
		Type:      ContestUpdateType,
		ContestID: contestId,
		UpdatedBy: updatedBy,
		Timestamp: time.Now(),
		Contest:   contestUpdate,
	}
}

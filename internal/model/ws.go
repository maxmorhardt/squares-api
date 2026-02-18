package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	SquareUpdateType        string = "square_update"
	ContestUpdateType       string = "contest_update"
	QuarterResultUpdateType string = "quarter_result_update"
	ContestDeletedType      string = "contest_deleted"
	ConnectedType           string = "connected"
	DisconnectType          string = "disconnected"
	ContestChannelPrefix    string = "contest"
)

type WSUpdate struct {
	Type          string         `json:"type"`
	ContestID     uuid.UUID      `json:"contestId"`
	ConnectionID  uuid.UUID      `json:"connectionId,omitempty"`
	UpdatedBy     string         `json:"updatedBy"`
	Timestamp     time.Time      `json:"timestamp"`
	Square        *Square        `json:"square,omitempty"`
	Contest       *Contest       `json:"contest,omitempty"`
	QuarterResult *QuarterResult `json:"quarterResult,omitempty"`
}

func NewConnectedMessage(connectionID uuid.UUID) *WSUpdate {
	return &WSUpdate{
		Type:         ConnectedType,
		ConnectionID: connectionID,
		UpdatedBy:    "system",
		Timestamp:    time.Now(),
	}
}

func NewDisconnectedMessage(connectionID uuid.UUID) *WSUpdate {
	return &WSUpdate{
		Type:         DisconnectType,
		ConnectionID: connectionID,
		UpdatedBy:    "system",
		Timestamp:    time.Now(),
	}
}

func NewSquareUpdateMessage(contestId uuid.UUID, updatedBy string, square *Square) *WSUpdate {
	return &WSUpdate{
		Type:      SquareUpdateType,
		ContestID: contestId,
		UpdatedBy: updatedBy,
		Timestamp: time.Now(),
		Square:    square,
	}
}

func NewQuarterResultUpdateMessage(contestId uuid.UUID, updatedBy string, quarterResult *QuarterResult) *WSUpdate {
	return &WSUpdate{
		Type:          QuarterResultUpdateType,
		ContestID:     contestId,
		UpdatedBy:     updatedBy,
		Timestamp:     time.Now(),
		QuarterResult: quarterResult,
	}
}

func NewContestUpdateMessage(contestId uuid.UUID, updatedBy string, contest *Contest) *WSUpdate {
	return &WSUpdate{
		Type:      ContestUpdateType,
		ContestID: contestId,
		UpdatedBy: updatedBy,
		Timestamp: time.Now(),
		Contest:   contest,
	}
}

func NewContestDeletedMessage(contestId uuid.UUID, updatedBy string) *WSUpdate {
	return &WSUpdate{
		Type:      ContestDeletedType,
		ContestID: contestId,
		UpdatedBy: updatedBy,
		Timestamp: time.Now(),
	}
}

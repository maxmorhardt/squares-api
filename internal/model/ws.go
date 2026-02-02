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
	Type          string                 `json:"type"`
	ContestID     uuid.UUID              `json:"contestId"`
	ConnectionID  uuid.UUID              `json:"connectionId,omitempty"`
	UpdatedBy     string                 `json:"updatedBy"`
	Timestamp     time.Time              `json:"timestamp"`
	Square        *SquareWSUpdate        `json:"square,omitempty"`
	Contest       *ContestWSUpdate       `json:"contest,omitempty"`
	QuarterResult *QuarterResultWSUpdate `json:"quarterResult,omitempty"`
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

type QuarterResultWSUpdate struct {
	Quarter         int           `json:"quarter"`
	HomeTeamScore   int           `json:"homeTeamScore"`
	AwayTeamScore   int           `json:"awayTeamScore"`
	WinnerRow       int           `json:"winnerRow"`
	WinnerCol       int           `json:"winnerCol"`
	Winner          string        `json:"winner"`
	WinnerName      string        `json:"winnerName"`
	Status          ContestStatus `json:"status"`
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

func NewQuarterResultUpdateMessage(contestId uuid.UUID, updatedBy string, quarterResultUpdate *QuarterResultWSUpdate) *WSUpdate {
	return &WSUpdate{
		Type:          QuarterResultUpdateType,
		ContestID:     contestId,
		UpdatedBy:     updatedBy,
		Timestamp:     time.Now(),
		QuarterResult: quarterResultUpdate,
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

func NewContestDeletedMessage(contestId uuid.UUID, updatedBy string) *WSUpdate {
	return &WSUpdate{
		Type:      ContestDeletedType,
		ContestID: contestId,
		UpdatedBy: updatedBy,
		Timestamp: time.Now(),
	}
}

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
	ParticipantRemovedType  string = "participant_removed"
	ParticipantAddedType    string = "participant_added"
	ChatMessageType         string = "chat_message"
	ConnectedType           string = "connected"
	DisconnectType          string = "disconnected"
	ContestChannelPrefix    string = "contest"
)

type WSChatMessage struct {
	Message string `json:"message"`
}

type WSUpdate struct {
	Type          string              `json:"type"`
	ContestID     uuid.UUID           `json:"contestId"`
	ConnectionID  uuid.UUID           `json:"connectionId,omitempty"`
	UpdatedBy     string              `json:"updatedBy"`
	Timestamp     time.Time           `json:"timestamp"`
	Square        *Square             `json:"square,omitempty"`
	Contest       *Contest            `json:"contest,omitempty"`
	QuarterResult *QuarterResult      `json:"quarterResult,omitempty"`
	Participant   *ContestParticipant `json:"participant,omitempty"`
	Message       string              `json:"message,omitempty"`
}

func NewConnectedMessage(contestID, connectionID uuid.UUID) *WSUpdate {
	return &WSUpdate{
		Type:         ConnectedType,
		ContestID:    contestID,
		ConnectionID: connectionID,
		UpdatedBy:    "system",
		Timestamp:    time.Now(),
	}
}

func NewDisconnectedMessage(contestID, connectionID uuid.UUID) *WSUpdate {
	return &WSUpdate{
		Type:         DisconnectType,
		ContestID:    contestID,
		ConnectionID: connectionID,
		UpdatedBy:    "system",
		Timestamp:    time.Now(),
	}
}

func NewSquareUpdateMessage(contestID uuid.UUID, updatedBy string, square *Square) *WSUpdate {
	return &WSUpdate{
		Type:      SquareUpdateType,
		ContestID: contestID,
		UpdatedBy: updatedBy,
		Timestamp: time.Now(),
		Square:    square,
	}
}

func NewQuarterResultUpdateMessage(contestID uuid.UUID, updatedBy string, quarterResult *QuarterResult) *WSUpdate {
	return &WSUpdate{
		Type:          QuarterResultUpdateType,
		ContestID:     contestID,
		UpdatedBy:     updatedBy,
		Timestamp:     time.Now(),
		QuarterResult: quarterResult,
	}
}

func NewContestUpdateMessage(contestID uuid.UUID, updatedBy string, contest *Contest) *WSUpdate {
	return &WSUpdate{
		Type:      ContestUpdateType,
		ContestID: contestID,
		UpdatedBy: updatedBy,
		Timestamp: time.Now(),
		Contest:   contest,
	}
}

func NewContestDeletedMessage(contestID uuid.UUID, updatedBy string) *WSUpdate {
	return &WSUpdate{
		Type:      ContestDeletedType,
		ContestID: contestID,
		UpdatedBy: updatedBy,
		Timestamp: time.Now(),
	}
}

func NewChatMessage(contestID uuid.UUID, sender, message string) *WSUpdate {
	return &WSUpdate{
		Type:      ChatMessageType,
		ContestID: contestID,
		UpdatedBy: sender,
		Timestamp: time.Now(),
		Message:   message,
	}
}

func NewParticipantRemovedMessage(contestID uuid.UUID, updatedBy string, participant *ContestParticipant) *WSUpdate {
	return &WSUpdate{
		Type:        ParticipantRemovedType,
		ContestID:   contestID,
		UpdatedBy:   updatedBy,
		Participant: participant,
		Timestamp:   time.Now(),
	}
}

func NewParticipantAddedMessage(contestID uuid.UUID, participant *ContestParticipant) *WSUpdate {
	return &WSUpdate{
		Type:        ParticipantAddedType,
		ContestID:   contestID,
		UpdatedBy:   participant.UserID,
		Participant: participant,
		Timestamp:   time.Now(),
	}
}

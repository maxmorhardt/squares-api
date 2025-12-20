package model

import "slices"

type ContestStatus string

const (
	ContestStatusActive   ContestStatus = "ACTIVE"
	ContestStatusQ1       ContestStatus = "Q1"
	ContestStatusQ2       ContestStatus = "Q2"
	ContestStatusQ3       ContestStatus = "Q3"
	ContestStatusQ4       ContestStatus = "Q4"
	ContestStatusFinished ContestStatus = "FINISHED"
	ContestStatusDeleted  ContestStatus = "DELETED"
)

func (cs ContestStatus) String() string {
	return string(cs)
}

func (cs ContestStatus) IsValid() bool {
	switch cs {
	case ContestStatusActive,
		ContestStatusQ1, ContestStatusQ2, ContestStatusQ3, ContestStatusQ4,
		ContestStatusFinished, ContestStatusDeleted:
		return true
	}
	return false
}

func (cs ContestStatus) CanTransitionTo(target ContestStatus) bool {
	if cs == target {
		return true
	}

	validTransitions := map[ContestStatus][]ContestStatus{
		ContestStatusActive:   {ContestStatusQ1},
		ContestStatusQ1:       {ContestStatusQ2},
		ContestStatusQ2:       {ContestStatusQ3},
		ContestStatusQ3:       {ContestStatusQ4},
		ContestStatusQ4:       {ContestStatusFinished},
		ContestStatusFinished: {},
		ContestStatusDeleted:  {},
	}

	allowedTargets, exists := validTransitions[cs]
	if !exists {
		return false
	}

	return slices.Contains(allowedTargets, target)
}

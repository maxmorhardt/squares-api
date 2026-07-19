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

func (cs ContestStatus) IsTerminal() bool {
	return cs == ContestStatusFinished || cs == ContestStatusDeleted
}

func (cs ContestStatus) Quarter() (int, bool) {
	switch cs {
	case ContestStatusQ1:
		return 1, true
	case ContestStatusQ2:
		return 2, true
	case ContestStatusQ3:
		return 3, true
	case ContestStatusQ4:
		return 4, true
	default:
		return 0, false
	}
}

func StatusAfterQuarter(quarter int) (ContestStatus, bool) {
	switch quarter {
	case 1:
		return ContestStatusQ2, true
	case 2:
		return ContestStatusQ3, true
	case 3:
		return ContestStatusQ4, true
	case 4:
		return ContestStatusFinished, true
	default:
		return "", false
	}
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

func PreviousQuarterStatus(cs ContestStatus) (status ContestStatus, quarter int, ok bool) {
	switch cs {
	case ContestStatusQ2:
		return ContestStatusQ1, 1, true
	case ContestStatusQ3:
		return ContestStatusQ2, 2, true
	case ContestStatusQ4:
		return ContestStatusQ3, 3, true
	case ContestStatusFinished:
		return ContestStatusQ4, 4, true
	default:
		return "", 0, false
	}
}

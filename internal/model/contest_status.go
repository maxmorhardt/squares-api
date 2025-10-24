package model

type ContestStatus string

const (
	ContestStatusActive     ContestStatus = "ACTIVE"
	ContestStatusLocked     ContestStatus = "LOCKED"
	ContestStatusQ1         ContestStatus = "Q1"
	ContestStatusQ2         ContestStatus = "Q2"
	ContestStatusQ3         ContestStatus = "Q3"
	ContestStatusQ4         ContestStatus = "Q4"
	ContestStatusFinished   ContestStatus = "FINISHED"
	ContestStatusCancelled  ContestStatus = "CANCELLED"
	ContestStatusDeleted    ContestStatus = "DELETED"
)

func (cs ContestStatus) String() string {
	return string(cs)
}

func (cs ContestStatus) IsValid() bool {
	switch cs {
	case ContestStatusActive, ContestStatusLocked,
		ContestStatusQ1, ContestStatusQ2, ContestStatusQ3, ContestStatusQ4,
		ContestStatusFinished, ContestStatusCancelled, ContestStatusDeleted:
		return true
	}
	return false
}

func AllContestStatuses() []ContestStatus {
	return []ContestStatus{
		ContestStatusActive,
		ContestStatusLocked,
		ContestStatusQ1,
		ContestStatusQ2,
		ContestStatusQ3,
		ContestStatusQ4,
		ContestStatusFinished,
		ContestStatusCancelled,
		ContestStatusDeleted,
	}
}

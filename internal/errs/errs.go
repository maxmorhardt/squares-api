package errs

import "errors"

var (
	ErrUnauthorizedContestEdit   = errors.New("only the contest owner can update this contest")
	ErrUnauthorizedContestDelete = errors.New("only the contest owner can delete this contest")
	ErrUnauthorizedSquareEdit    = errors.New("only the square owner can update this square")
)

var (
	ErrInvalidContestName  = errors.New("contest name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	ErrInvalidHomeTeamName = errors.New("home team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	ErrInvalidAwayTeamName = errors.New("away team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	ErrInvalidSquareValue  = errors.New("value must be 1-3 uppercase letters or numbers")
)

var (
	ErrContestNotEditable   = errors.New("contest is not in an editable state")
	ErrSquareNotEditable    = errors.New("squares can only be edited when contest is active")
	ErrContestAlreadyExists = errors.New("contest already exists with this name")
)

var (
	ErrDatabaseUnavailable = errors.New("service temporarily unavailable, please try again later")
)

var (
	ErrContestNotFound    = errors.New("contest not found")
	ErrSquareNotFound     = errors.New("square not found")
	ErrInvalidRequestBody = errors.New("invalid request body")
	ErrClaimsNotFound     = errors.New("authentication required")
)

var (
	ErrInvalidPage  = errors.New("invalid page parameter")
	ErrInvalidLimit = errors.New("invalid limit parameter")
)

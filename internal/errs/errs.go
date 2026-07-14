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
	ErrContestNotEditable         = errors.New("contest is not in an editable state")
	ErrContestFinalized           = errors.New("contest is finished or deleted and cannot be modified")
	ErrContestNotReady            = errors.New("all squares must be claimed before the contest can be started")
	ErrSquareNotEditable          = errors.New("squares can only be edited when contest is active")
	ErrContestAlreadyExists       = errors.New("contest already exists with this name")
	ErrQuarterResultAlreadyExists = errors.New("result of this quarter has already been recorded")
)

var (
	ErrDatabaseUnavailable = errors.New("service temporarily unavailable, please try again later")
)

var (
	ErrContestNotFound       = errors.New("contest not found")
	ErrSquareNotFound        = errors.New("square not found")
	ErrUserNotFound          = errors.New("user not found")
	ErrAccountActiveContests = errors.New("you must delete or leave your active contests before deleting your account")
	ErrInvalidRequestBody    = errors.New("invalid request body")
	ErrClaimsNotFound        = errors.New("authentication required")
	ErrClaimsParse           = errors.New("claims parse failed")
	ErrEmailUnverified       = errors.New("token has no verified email")
)

var (
	ErrInvalidPage  = errors.New("invalid page parameter")
	ErrInvalidLimit = errors.New("invalid limit parameter")
)

var (
	ErrInvalidTurnstile      = errors.New("invalid or expired captcha")
	ErrTurnstileVerification = errors.New("failed to verify turnstile token")
	ErrEmailNotification     = errors.New("failed to send contact email notification")
)

var (
	ErrInviteNotFound       = errors.New("invite not found")
	ErrInviteExpired        = errors.New("invite link has expired")
	ErrInviteMaxUsesReached = errors.New("invite link has reached its usage limit")
	ErrNotEnoughSquares     = errors.New("not enough squares remaining in this contest")
	ErrAlreadyParticipant   = errors.New("you are already a participant in this contest")
	ErrNotParticipant       = errors.New("not a participant in this contest")
	ErrInsufficientRole     = errors.New("insufficient permissions for this action")
	ErrCannotRemoveOwner    = errors.New("cannot remove the contest owner")
	ErrCannotChangeOwner    = errors.New("cannot change the owner's role")
	ErrSquareLimitReached   = errors.New("you have reached your square limit for this contest")
	ErrSquareLimitTooLow    = errors.New("new limit cannot be below the number of squares already claimed")
	ErrInvalidSquareCount   = errors.New("participant invites must grant at least one square")
)

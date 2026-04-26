package model

const (
	SquaresAdminGroup string = "squares-admin"
)

type Claims struct {
	Username string   `json:"preferred_username"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
	Name     string   `json:"given_name"`
	LastName string   `json:"family_name"`
	Expire   int64    `json:"exp"`
	IssuedAt int64    `json:"iat"`
}

type AuthFailureReason string

const (
	AuthFailureMissingHeader AuthFailureReason = "missing_header"
	AuthFailureVerifyFailed  AuthFailureReason = "verify_failed"
	AuthFailureClaimsParse   AuthFailureReason = "claims_parse_failed"
)

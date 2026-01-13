package model

const (
	SquaresAdminGroup string = "squares-admin"
)

type Claims struct {
	Username  string   `json:"preferred_username"`
	Groups    []string `json:"groups"`
	FirstName string   `json:"given_name"`
	LastName  string   `json:"family_name"`
	Email     string   `json:"email"`
	Scope     string   `json:"scope"`
	Expire    int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	Subject   string   `json:"sub"`
}
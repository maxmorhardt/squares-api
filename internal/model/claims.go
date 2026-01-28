package model

const (
	SquaresAdminRole string = "squares-admin"
)

type Roles map[string]map[string]string

type Claims struct {
	Subject   string `json:"sub"`
	Email     string `json:"email"`
	Roles     Roles  `json:"urn:zitadel:iam:org:project:roles"`
	FirstName string `json:"given_name"`
	LastName  string `json:"family_name"`
	Expire    int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}
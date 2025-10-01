package auth

type Claims struct {
	Username string   `json:"preferred_username"`
	Roles    []string `json:"roles"`
}
package model

type ctxKey string

const (
	UserKey      ctxKey = "user"
	RequestIDKey ctxKey = "request_id"
	RolesKey     ctxKey = "roles"
	LoggerKey    ctxKey = "logger"
)
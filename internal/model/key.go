package model

type ctxKey string

const (
	UserKey      ctxKey = "user"
	RequestIDKey ctxKey = "request_id"
	LoggerKey    ctxKey = "logger"
	ClaimsKey    ctxKey = "claims"
)

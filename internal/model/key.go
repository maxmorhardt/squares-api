package model

type CTXKey string

const (
	UserKey         CTXKey = "user"
	RequestIDKey    CTXKey = "request_id"
	LoggerKey       CTXKey = "logger"
	ClaimsKey       CTXKey = "claims"
	ConnectionIDKey CTXKey = "connection_id"
)

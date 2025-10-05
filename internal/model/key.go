package model

type ctxKey string

const UserKey ctxKey = "user"
const RequestIDKey ctxKey = "request_id"
const RolesKey ctxKey = "roles"
const LoggerKey ctxKey = "logger"
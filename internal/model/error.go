package model

import (
	"time"

	"github.com/gin-gonic/gin"
)

type APIError struct {
	Code      int       `json:"code" example:"400"`
	Message   string    `json:"message" example:"invalid request"`
	Timestamp time.Time `json:"timestamp" example:"2025-10-05T13:45:00Z"`
	RequestID string    `json:"requestId" example:"7db692fa-c767-468f-af3d-9231b0f88c69"`
}

func NewAPIError(code int, message string, c *gin.Context) APIError {
	return APIError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		RequestID: c.GetString(RequestIDKey),
	}
}
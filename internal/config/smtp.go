package config

import (
	"log/slog"
	"os"
)

var (
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SupportEmail string
)

const errorMessage = "incomplete smtp configuration in environment variables"

func InitSMTP() {
	SMTPHost = os.Getenv("SMTP_HOST")
	SMTPPort = os.Getenv("SMTP_PORT")
	SMTPUser = os.Getenv("SMTP_USER")
	SMTPPassword = os.Getenv("SMTP_PASSWORD")
	SupportEmail = os.Getenv("SUPPORT_EMAIL")

	if SMTPHost == "" || SMTPPort == "" || SMTPUser == "" || SMTPPassword == "" || SupportEmail == "" {
		slog.Error(errorMessage)
		panic(errorMessage)
	}
}
package service

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type ContactService interface {
	SubmitContact(ctx context.Context, req *model.ContactRequest, ipAddress string) error
}

type contactService struct {
	repo repository.ContactRepository
}

func NewContactService(repo repository.ContactRepository) ContactService {
	return &contactService{
		repo: repo,
	}
}

func (s *contactService) SubmitContact(ctx context.Context, req *model.ContactRequest, ipAddress string) error {
	log := util.LoggerFromContext(ctx)

	// create contact submission record
	submission := &model.ContactSubmission{
		Name:      req.Name,
		Email:     req.Email,
		Subject:   req.Subject,
		Message:   req.Message,
		IPAddress: ipAddress,
		Status:    "pending",
	}

	// save to database
	if err := s.repo.Create(ctx, submission); err != nil {
		log.Error("failed to save contact submission to database", "error", err)
		return err
	}

	log.Info("contact submission saved to database", "submission_id", submission.ID)

	// send email notification asynchronously
	go func() {
		if err := s.sendEmailNotification(req); err != nil {
			log.Error("failed to send contact email notification", "error", err, "submission_id", submission.ID)
		} else {
			log.Info("contact email notification sent", "submission_id", submission.ID)
		}
	}()

	return nil
}

func (s *contactService) sendEmailNotification(req *model.ContactRequest) error {
	// construct email message
	from := config.SMTPUser
	to := []string{config.SupportEmail}
	subject := fmt.Sprintf("Contact Form: %s", req.Subject)
	body := fmt.Sprintf(
		"New contact form submission:\n\n"+
			"Name: %s\n"+
			"Email: %s\n"+
			"Subject: %s\n\n"+
			"Message:\n%s\n",
		req.Name,
		req.Email,
		req.Subject,
		req.Message,
	)

	message := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"Reply-To: %s\r\n"+
			"\r\n"+
			"%s",
		from,
		config.SupportEmail,
		subject,
		req.Email,
		body,
	)

	// send email via smtp
	auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPassword, config.SMTPHost)
	addr := fmt.Sprintf("%s:%s", config.SMTPHost, config.SMTPPort)

	return smtp.SendMail(addr, auth, from, to, []byte(message))
}

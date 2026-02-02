package service

import (
	"context"
	"fmt"
	"net/smtp"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/errs"
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

	// validate turnstile token
	if err := s.validateTurnstile(ctx, req.TurnstileToken, ipAddress); err != nil {
		log.Error("turnstile validation failed", "error", err)
		return errs.ErrInvalidTurnstile
	}

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

type turnstileResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
	Action      string   `json:"action"`
	CData       string   `json:"cdata"`
}

func (s *contactService) validateTurnstile(ctx context.Context, token, remoteIP string) error {
	client := resty.New().
		SetTimeout(10 * time.Second).
		SetBaseURL("https://challenges.cloudflare.com")

	var turnstileResp turnstileResponse
	resp, err := client.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"secret":   config.TurnstileSecretKey,
			"response": token,
			"remoteip": remoteIP,
		}).
		SetResult(&turnstileResp).
		Post("/turnstile/v0/siteverify")

	if err != nil {
		return fmt.Errorf("failed to verify turnstile token: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("turnstile API returned status %d", resp.StatusCode())
	}

	if !turnstileResp.Success {
		return fmt.Errorf("turnstile validation failed: %v", turnstileResp.ErrorCodes)
	}

	return nil
}

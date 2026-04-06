package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/smtp"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/templates"
	"github.com/maxmorhardt/squares-api/internal/util"
)

var contactEmailTmpl = template.Must(template.New("contact_email").Parse(templates.ContactEmailHTML))

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
	from := config.Env().SMTP.User
	to := []string{config.Env().SMTP.SupportEmail}
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

	// render HTML body
	var htmlBody bytes.Buffer
	if err := contactEmailTmpl.Execute(&htmlBody, req); err != nil {
		return fmt.Errorf("failed to render contact email template: %w", err)
	}

	message := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"Reply-To: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: multipart/alternative; boundary=\"boundary42\"\r\n"+
			"\r\n"+
			"--boundary42\r\n"+
			"Content-Type: text/plain; charset=\"UTF-8\"\r\n"+
			"\r\n"+
			"%s\r\n"+
			"--boundary42\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n"+
			"\r\n"+
			"%s\r\n"+
			"--boundary42--",
		from,
		config.Env().SMTP.SupportEmail,
		subject,
		req.Email,
		body,
		htmlBody.String(),
	)

	// send email via smtp
	auth := smtp.PlainAuth("", config.Env().SMTP.User, config.Env().SMTP.Password, config.Env().SMTP.Host)
	addr := fmt.Sprintf("%s:%d", config.Env().SMTP.Host, config.Env().SMTP.Port)

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
			"secret":   config.Env().Turnstile.SecretKey,
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

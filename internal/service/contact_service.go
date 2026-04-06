package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"
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

// sanitizeHeader strips CR and LF characters to prevent email header injection.
func sanitizeHeader(v string) string {
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", "")
	return v
}

func (s *contactService) sendEmailNotification(req *model.ContactRequest) error {
	// construct email message
	from := config.Env().SMTP.User
	to := []string{config.Env().SMTP.SupportEmail}
	subject := sanitizeHeader(fmt.Sprintf("Contact Form: %s", req.Subject))
	plainBody := fmt.Sprintf(
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

	// build message with unique multipart boundary
	var msg bytes.Buffer

	// write top-level headers
	fmt.Fprintf(&msg, "From: %s\r\n", sanitizeHeader(from))
	fmt.Fprintf(&msg, "To: %s\r\n", sanitizeHeader(config.Env().SMTP.SupportEmail))
	fmt.Fprintf(&msg, "Subject: %s\r\n", subject)
	fmt.Fprintf(&msg, "Reply-To: %s\r\n", sanitizeHeader(req.Email))
	fmt.Fprintf(&msg, "MIME-Version: 1.0\r\n")

	mpw := multipart.NewWriter(&msg)
	fmt.Fprintf(&msg, "Content-Type: multipart/alternative; boundary=%q\r\n", mpw.Boundary())
	fmt.Fprintf(&msg, "\r\n")

	// plain text part
	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Type", `text/plain; charset="UTF-8"`)
	pw, err := mpw.CreatePart(partHeader)
	if err != nil {
		return fmt.Errorf("failed to create plain text MIME part: %w", err)
	}
	pw.Write([]byte(plainBody))

	// HTML part
	partHeader = make(textproto.MIMEHeader)
	partHeader.Set("Content-Type", `text/html; charset="UTF-8"`)
	pw, err = mpw.CreatePart(partHeader)
	if err != nil {
		return fmt.Errorf("failed to create HTML MIME part: %w", err)
	}
	pw.Write(htmlBody.Bytes())

	mpw.Close()

	// send email via smtp
	auth := smtp.PlainAuth("", config.Env().SMTP.User, config.Env().SMTP.Password, config.Env().SMTP.Host)
	addr := fmt.Sprintf("%s:%d", config.Env().SMTP.Host, config.Env().SMTP.Port)

	return smtp.SendMail(addr, auth, from, to, msg.Bytes())
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

package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
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
	cfg  *config.Config
}

func NewContactService(repo repository.ContactRepository, cfg *config.Config) ContactService {
	return &contactService{
		repo: repo,
		cfg:  cfg,
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
		Status:    model.ContactStatusPending,
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

func sanitizeHeader(v string) string {
	v = strings.ReplaceAll(v, "\r", "")
	v = strings.ReplaceAll(v, "\n", "")
	return v
}

func (s *contactService) sendEmailNotification(req *model.ContactRequest) error {
	msg, err := s.buildContactEmail(req)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrEmailNotification, err)
	}

	auth := smtp.PlainAuth("", s.cfg.SMTP.User, s.cfg.SMTP.Password, s.cfg.SMTP.Host)
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTP.Host, s.cfg.SMTP.Port)
	to := []string{s.cfg.SMTP.SupportEmail}

	if err := smtp.SendMail(addr, auth, s.cfg.SMTP.User, to, msg); err != nil {
		return fmt.Errorf("%w: %w", errs.ErrEmailNotification, err)
	}

	return nil
}

func (s *contactService) buildContactEmail(req *model.ContactRequest) ([]byte, error) {
	var htmlBody bytes.Buffer
	if err := contactEmailTmpl.Execute(&htmlBody, req); err != nil {
		return nil, fmt.Errorf("failed to render contact email template: %w", err)
	}

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

	var msg bytes.Buffer

	// top-level headers
	fmt.Fprintf(&msg, "From: %s\r\n", sanitizeHeader(s.cfg.SMTP.User))
	fmt.Fprintf(&msg, "To: %s\r\n", sanitizeHeader(s.cfg.SMTP.SupportEmail))
	fmt.Fprintf(&msg, "Subject: %s\r\n", sanitizeHeader(fmt.Sprintf("Contact Form: %s", req.Subject)))
	fmt.Fprintf(&msg, "Reply-To: %s\r\n", sanitizeHeader(req.Email))
	fmt.Fprintf(&msg, "MIME-Version: 1.0\r\n")

	mpw := multipart.NewWriter(&msg)
	fmt.Fprintf(&msg, "Content-Type: multipart/alternative; boundary=%q\r\n", mpw.Boundary())
	fmt.Fprintf(&msg, "\r\n")

	if err := writeMIMEPart(mpw, `text/plain; charset="UTF-8"`, []byte(plainBody)); err != nil {
		return nil, err
	}
	if err := writeMIMEPart(mpw, `text/html; charset="UTF-8"`, htmlBody.Bytes()); err != nil {
		return nil, err
	}

	if err := mpw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return msg.Bytes(), nil
}

func writeMIMEPart(mpw *multipart.Writer, contentType string, body []byte) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", contentType)

	pw, err := mpw.CreatePart(header)
	if err != nil {
		return fmt.Errorf("failed to create MIME part %q: %w", contentType, err)
	}
	if _, err := pw.Write(body); err != nil {
		return fmt.Errorf("failed to write MIME part %q: %w", contentType, err)
	}

	return nil
}

func (s *contactService) validateTurnstile(ctx context.Context, token, remoteIP string) error {
	baseURL := s.cfg.Turnstile.BaseURL
	if baseURL == "" {
		baseURL = "https://challenges.cloudflare.com"
	}

	client := resty.New().
		SetTimeout(10 * time.Second).
		SetBaseURL(baseURL).
		SetRetryCount(2).
		SetRetryWaitTime(200 * time.Millisecond).
		SetRetryMaxWaitTime(1 * time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			if ctx.Err() != nil {
				return false
			}
			if err != nil {
				return true
			}
			return r.StatusCode() >= http.StatusInternalServerError
		})

	var turnstileResp model.TurnstileResponse
	resp, err := client.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"secret":   s.cfg.Turnstile.SecretKey,
			"response": token,
			"remoteip": remoteIP,
		}).
		SetResult(&turnstileResp).
		Post("/turnstile/v0/siteverify")

	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrTurnstileVerification, err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("%w: status %d", errs.ErrTurnstileVerification, resp.StatusCode())
	}

	if !turnstileResp.Success {
		return fmt.Errorf("%w: %v", errs.ErrInvalidTurnstile, turnstileResp.ErrorCodes)
	}

	return nil
}

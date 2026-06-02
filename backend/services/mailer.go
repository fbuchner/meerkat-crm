package services

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"

	"meerkat/config"
	"meerkat/logger"

	"github.com/resend/resend-go/v2"
)

// EmailMessage is a transport-agnostic, already-rendered email ready for delivery.
type EmailMessage struct {
	To      string
	Subject string
	HTML    string
}

// SendEmail delivers msg through every configured channel (Resend and/or SMTP).
// Delivery is best-effort: returns nil if at least one configured channel
// succeeds, and a combined error only if all configured channels fail. If no
// channel is configured it logs a warning and returns nil (no-op)
func SendEmail(cfg config.Config, msg EmailMessage) error {
	if msg.To == "" {
		logger.Warn().Msg("Skipping email because recipient address is empty")
		return nil
	}

	if !cfg.EmailEnabled() {
		logger.Warn().Str("to", msg.To).Msg("No email channel configured; email not sent")
		return nil
	}

	var (
		attempted int
		succeeded int
		errs      []string
	)

	if cfg.UseResend {
		attempted++
		if err := sendViaResend(cfg, msg); err != nil {
			logger.Error().Err(err).Str("to", msg.To).Msg("Failed to send email via Resend")
			errs = append(errs, fmt.Sprintf("resend: %v", err))
		} else {
			succeeded++
		}
	}

	if cfg.UseSMTP {
		attempted++
		if err := sendViaSMTP(cfg, msg); err != nil {
			logger.Error().Err(err).Str("to", msg.To).Msg("Failed to send email via SMTP")
			errs = append(errs, fmt.Sprintf("smtp: %v", err))
		} else {
			succeeded++
		}
	}

	if succeeded == 0 {
		return fmt.Errorf("all email channels failed (%d attempted): %s", attempted, strings.Join(errs, "; "))
	}

	if len(errs) > 0 {
		logger.Warn().Str("to", msg.To).Int("succeeded", succeeded).Int("attempted", attempted).Msg("Email delivered on some but not all channels")
	}

	return nil
}

// delivers the message through the Resend API.
func sendViaResend(cfg config.Config, msg EmailMessage) error {
	client := resend.NewClient(cfg.ResendAPIKey)

	params := &resend.SendEmailRequest{
		From:    cfg.ResendFromEmail,
		To:      []string{msg.To},
		Subject: msg.Subject,
		Html:    msg.HTML,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return err
	}

	logger.Info().Str("email_id", sent.Id).Str("to", msg.To).Msg("Email sent via Resend")
	return nil
}

// delivers the message through an SMTP server
func sendViaSMTP(cfg config.Config, msg EmailMessage) error {
	addr := net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort))
	body := buildSMTPMessage(cfg.SMTPFromEmail, msg)

	var auth smtp.Auth
	if cfg.SMTPUsername != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
	}

	if cfg.SMTPUseTLS {
		if err := sendSMTPImplicitTLS(cfg, addr, auth, msg.To, body); err != nil {
			return err
		}
	} else {
		// smtp.SendMail upgrades to STARTTLS automatically when the server advertises it.
		if err := smtp.SendMail(addr, auth, cfg.SMTPFromEmail, []string{msg.To}, body); err != nil {
			return err
		}
	}

	logger.Info().Str("to", msg.To).Msg("Email sent via SMTP")
	return nil
}

// sendSMTPImplicitTLS sends a message over a connection that is wrapped in TLS (implicit TLS, typically port 465).
func sendSMTPImplicitTLS(cfg config.Config, addr string, auth smtp.Auth, to string, body []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.SMTPHost})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}

	client, err := smtp.NewClient(conn, cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(cfg.SMTPFromEmail); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp data close: %w", err)
	}

	return client.Quit()
}

// buildSMTPMessage assembles a minimal RFC 5322 HTML email.
func buildSMTPMessage(from string, msg EmailMessage) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + msg.To + "\r\n")
	b.WriteString("Subject: " + mime.QEncoding.Encode("UTF-8", msg.Subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(msg.HTML)
	return []byte(b.String())
}

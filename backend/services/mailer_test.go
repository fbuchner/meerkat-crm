package services

import (
	"bufio"
	"net"
	"strings"
	"testing"

	"mycorrhizal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startFakeSMTPServer starts a minimal single-connection SMTP server on an
// ephemeral local port and returns its host/port. It speaks just enough of
// the protocol (EHLO/HELO, AUTH PLAIN, MAIL FROM, RCPT TO, DATA, QUIT) for
// net/smtp.SendMail to complete a full send, so tests can exercise
// sendViaSMTP's success path without any live network access. The server
// handles exactly one connection and then exits.
func startFakeSMTPServer(t *testing.T) (host string, port int) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		respond := func(line string) {
			_, _ = conn.Write([]byte(line + "\r\n"))
		}

		respond("220 localhost ESMTP fake")
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				respond("250-localhost")
				respond("250 AUTH PLAIN")
			case strings.HasPrefix(cmd, "AUTH"):
				respond("235 2.7.0 Authentication successful")
			case strings.HasPrefix(cmd, "MAIL FROM"):
				respond("250 OK")
			case strings.HasPrefix(cmd, "RCPT TO"):
				respond("250 OK")
			case strings.HasPrefix(cmd, "DATA"):
				respond("354 Start mail input")
				for {
					dataLine, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if strings.TrimRight(dataLine, "\r\n") == "." {
						break
					}
				}
				respond("250 OK")
			case strings.HasPrefix(cmd, "QUIT"):
				respond("221 Bye")
				return
			default:
				respond("250 OK")
			}
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port
}

// TestSendEmail_NoRecipient verifies SendEmail short-circuits with a nil
// (no-op) return when the recipient address is empty, before any channel is
// consulted.
func TestSendEmail_NoRecipient(t *testing.T) {
	cfg := config.Config{UseSMTP: true, SMTPHost: "127.0.0.1", SMTPPort: 1, SMTPFromEmail: "noreply@example.com"}
	err := SendEmail(cfg, EmailMessage{To: "", Subject: "hi", HTML: "<p>hi</p>"})
	assert.NoError(t, err)
}

// TestSendEmail_NoChannelConfigured verifies SendEmail no-ops successfully
// when neither Resend nor SMTP is configured.
func TestSendEmail_NoChannelConfigured(t *testing.T) {
	cfg := config.Config{}
	err := SendEmail(cfg, EmailMessage{To: "user@example.com", Subject: "hi", HTML: "<p>hi</p>"})
	assert.NoError(t, err)
}

// TestSendEmail_SMTPFailure verifies SendEmail returns a combined error that
// mentions the smtp channel when the only configured channel fails.
func TestSendEmail_SMTPFailure(t *testing.T) {
	cfg := config.Config{
		UseSMTP:       true,
		SMTPHost:      "127.0.0.1",
		SMTPPort:      1, // nothing listens here: connection refused, fast failure
		SMTPFromEmail: "noreply@example.com",
	}
	err := SendEmail(cfg, EmailMessage{To: "user@example.com", Subject: "hi", HTML: "<p>hi</p>"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "smtp:")
	assert.Contains(t, err.Error(), "all email channels failed")
}

// TestSendEmail_SMTPSuccess verifies SendEmail returns nil when the only
// configured channel (SMTP) succeeds end-to-end against a fake local server.
func TestSendEmail_SMTPSuccess(t *testing.T) {
	host, port := startFakeSMTPServer(t)
	cfg := config.Config{
		UseSMTP:       true,
		SMTPHost:      host,
		SMTPPort:      port,
		SMTPFromEmail: "noreply@example.com",
	}
	err := SendEmail(cfg, EmailMessage{To: "user@example.com", Subject: "hi", HTML: "<p>hi</p>"})
	assert.NoError(t, err)
}

// TestSendViaSMTP_Success verifies sendViaSMTP completes a full plaintext
// (non-TLS) send against a fake local SMTP server, without authentication.
func TestSendViaSMTP_Success(t *testing.T) {
	host, port := startFakeSMTPServer(t)
	cfg := config.Config{
		SMTPHost:      host,
		SMTPPort:      port,
		SMTPFromEmail: "noreply@example.com",
	}
	err := sendViaSMTP(cfg, EmailMessage{To: "user@example.com", Subject: "hi", HTML: "<p>hi</p>"})
	assert.NoError(t, err)
}

// TestSendViaSMTP_SuccessWithAuth verifies sendViaSMTP exercises the
// smtp.PlainAuth code path (cfg.SMTPUsername set) against a fake local
// server. PlainAuth permits unencrypted connections when the server name is
// "127.0.0.1"/"localhost", so this succeeds without needing real TLS.
func TestSendViaSMTP_SuccessWithAuth(t *testing.T) {
	host, port := startFakeSMTPServer(t)
	cfg := config.Config{
		SMTPHost:      host,
		SMTPPort:      port,
		SMTPFromEmail: "noreply@example.com",
		SMTPUsername:  "smtpuser",
		SMTPPassword:  "smtppass",
	}
	err := sendViaSMTP(cfg, EmailMessage{To: "user@example.com", Subject: "hi", HTML: "<p>hi</p>"})
	assert.NoError(t, err)
}

// TestSendViaSMTP_ConnectionRefused verifies sendViaSMTP propagates the
// underlying error when the plaintext (non-TLS) connection cannot be
// established. Reuses the closed-local-port idiom from
// password_reset_service_test.go to fail fast without live network access.
func TestSendViaSMTP_ConnectionRefused(t *testing.T) {
	cfg := config.Config{
		SMTPHost:      "127.0.0.1",
		SMTPPort:      1,
		SMTPFromEmail: "noreply@example.com",
	}
	err := sendViaSMTP(cfg, EmailMessage{To: "user@example.com", Subject: "hi", HTML: "<p>hi</p>"})
	require.Error(t, err)
}

// TestSendViaSMTP_ImplicitTLSConnectionRefused verifies sendViaSMTP routes
// through sendSMTPImplicitTLS when SMTPUseTLS is set, and propagates its
// error when the connection cannot be established.
func TestSendViaSMTP_ImplicitTLSConnectionRefused(t *testing.T) {
	cfg := config.Config{
		SMTPHost:      "127.0.0.1",
		SMTPPort:      1,
		SMTPFromEmail: "noreply@example.com",
		SMTPUseTLS:    true,
	}
	err := sendViaSMTP(cfg, EmailMessage{To: "user@example.com", Subject: "hi", HTML: "<p>hi</p>"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tls dial:")
}

// TestSendSMTPImplicitTLS_DialFailure directly unit-tests
// sendSMTPImplicitTLS's error wrapping when the TLS dial fails. A real TLS
// handshake success path is not testable here without a production-code
// change: the function hardcodes a tls.Config with no way to inject a
// trusted RootCAs pool for a self-signed test certificate, so a genuine
// success/post-dial branch has no test seam (same class of limitation as
// sendViaResend).
func TestSendSMTPImplicitTLS_DialFailure(t *testing.T) {
	err := sendSMTPImplicitTLS(config.Config{SMTPHost: "127.0.0.1"}, "127.0.0.1:1", nil, "user@example.com", []byte("body"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tls dial:")
}

// TestBuildSMTPMessage verifies the RFC 5322 message assembly: headers are
// present, the subject is Q-encoded (so non-ASCII/space-safe), and the HTML
// body is appended verbatim after the blank-line separator.
func TestBuildSMTPMessage(t *testing.T) {
	t.Run("assembles expected headers and body", func(t *testing.T) {
		msg := EmailMessage{To: "recipient@example.com", Subject: "Hello There", HTML: "<p>Hi</p>"}
		body := string(buildSMTPMessage("sender@example.com", msg))

		assert.Contains(t, body, "From: sender@example.com\r\n")
		assert.Contains(t, body, "To: recipient@example.com\r\n")
		assert.Contains(t, body, "MIME-Version: 1.0\r\n")
		assert.Contains(t, body, "Content-Type: text/html; charset=\"UTF-8\"\r\n")
		assert.Contains(t, body, "\r\n\r\n<p>Hi</p>")
		assert.True(t, strings.HasSuffix(body, "<p>Hi</p>"))
	})

	t.Run("Q-encodes a non-ASCII subject", func(t *testing.T) {
		msg := EmailMessage{To: "recipient@example.com", Subject: "Bienvenüe", HTML: "<p>Hi</p>"}
		body := string(buildSMTPMessage("sender@example.com", msg))

		assert.Contains(t, body, "Subject: =?UTF-8?q?")
		assert.NotContains(t, body, "Subject: Bienvenüe")
	})

	t.Run("leaves a plain ASCII subject unencoded", func(t *testing.T) {
		msg := EmailMessage{To: "recipient@example.com", Subject: "Plain Subject", HTML: "<p>Hi</p>"}
		body := string(buildSMTPMessage("sender@example.com", msg))

		assert.Contains(t, body, "Subject: Plain Subject\r\n")
	})
}

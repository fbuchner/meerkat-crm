package services

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"meerkat/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testEmail = EmailMessage{To: "friend@example.com", Subject: "Hello", HTML: "<p>Hi there</p>"}

// fakeResend stands in for the Resend API. resend-go offers no way to inject a transport via
// NewClient, but its default client leaves Transport nil, so it goes through
// http.DefaultTransport, which is swapped for the duration of the test. Tests in this package
// must not be parallel.
type fakeResend struct {
	mu       sync.Mutex
	requests []*http.Request
	bodies   []string
	respond  func() (*http.Response, error)
}

func useFakeResend(t *testing.T, respond func() (*http.Response, error)) *fakeResend {
	t.Helper()
	fake := &fakeResend{respond: respond}
	original := http.DefaultTransport
	http.DefaultTransport = fake
	t.Cleanup(func() { http.DefaultTransport = original })
	return fake
}

func (f *fakeResend) RoundTrip(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, _ := io.ReadAll(req.Body)
	f.requests = append(f.requests, req)
	f.bodies = append(f.bodies, string(body))
	return f.respond()
}

func (f *fakeResend) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func resendResponse(status int, body string) func() (*http.Response, error) {
	return func() (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Status:     http.StatusText(status),
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}
}

var (
	resendAccepts = resendResponse(http.StatusOK, `{"id":"email-1"}`)
	resendRejects = resendResponse(http.StatusUnprocessableEntity, `{"message":"invalid from address"}`)
)

// fakeSMTP is a just-enough SMTP server on loopback. rejectOn makes it refuse a command, named
// by its verb ("MAIL", "RCPT", "DATA", "AUTH").
type fakeSMTP struct {
	listener  net.Listener
	rejectOn  string
	startTLS  *tls.Config
	mu        sync.Mutex
	authPlain []string
	mailFrom  []string
	rcptTo    []string
	data      []string
	wg        sync.WaitGroup
}

func startFakeSMTP(t *testing.T, configure func(*fakeSMTP)) *fakeSMTP {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := &fakeSMTP{listener: listener}
	if configure != nil {
		configure(s)
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				defer conn.Close()
				s.serve(conn)
			}()
		}
	}()
	t.Cleanup(func() {
		listener.Close()
		s.wg.Wait()
	})
	return s
}

func (s *fakeSMTP) serve(conn net.Conn) {
	r := bufio.NewReader(conn)
	reply := func(line string) { io.WriteString(conn, line+"\r\n") }
	reply("220 fake ESMTP")

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		verb, arg, _ := strings.Cut(line, " ")
		verb = strings.ToUpper(verb)

		if verb == s.rejectOn {
			if verb == "AUTH" {
				reply("535 authentication failed")
			} else {
				reply("550 rejected")
			}
			continue
		}

		s.mu.Lock()
		switch verb {
		case "EHLO", "HELO":
			if s.startTLS != nil {
				reply("250-fake")
				reply("250-STARTTLS")
			} else {
				reply("250-fake")
			}
			reply("250 AUTH PLAIN")
		case "STARTTLS":
			reply("220 go ahead")
			s.mu.Unlock()
			tlsConn := tls.Server(conn, s.startTLS)
			if err := tlsConn.Handshake(); err == nil {
				s.serve(tlsConn)
			}
			return
		case "AUTH":
			_, encoded, _ := strings.Cut(arg, " ")
			decoded, _ := base64.StdEncoding.DecodeString(encoded)
			s.authPlain = append(s.authPlain, string(decoded))
			reply("235 ok")
		case "MAIL":
			s.mailFrom = append(s.mailFrom, arg)
			reply("250 ok")
		case "RCPT":
			s.rcptTo = append(s.rcptTo, arg)
			reply("250 ok")
		case "DATA":
			reply("354 go ahead")
			var b strings.Builder
			for {
				dataLine, err := r.ReadString('\n')
				if err != nil {
					s.mu.Unlock()
					return
				}
				if dataLine == ".\r\n" {
					break
				}
				b.WriteString(dataLine)
			}
			s.data = append(s.data, b.String())
			reply("250 queued")
		case "QUIT":
			reply("221 bye")
			s.mu.Unlock()
			return
		default:
			reply("250 ok")
		}
		s.mu.Unlock()
	}
}

func (s *fakeSMTP) delivered() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.data...)
}

// envelope returns what the server has seen so far besides message data.
func (s *fakeSMTP) envelope() (authPlain, mailFrom, rcptTo []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.authPlain...), append([]string(nil), s.mailFrom...), append([]string(nil), s.rcptTo...)
}

func (s *fakeSMTP) config() config.Config {
	addr := s.listener.Addr().(*net.TCPAddr)
	return config.Config{
		UseSMTP:       true,
		SMTPHost:      addr.IP.String(),
		SMTPPort:      addr.Port,
		SMTPFromEmail: "crm@example.com",
	}
}

// closedSMTPConfig points at a port nothing is listening on.
func closedSMTPConfig(t *testing.T) config.Config {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &fakeSMTP{listener: listener}
	cfg := s.config()
	listener.Close()
	return cfg
}

func resendConfig() config.Config {
	return config.Config{UseResend: true, ResendAPIKey: "re_test_key", ResendFromEmail: "crm@example.com"}
}

func withBothChannels(smtpCfg config.Config) config.Config {
	cfg := smtpCfg
	cfg.UseResend = true
	cfg.ResendAPIKey = "re_test_key"
	cfg.ResendFromEmail = "crm@example.com"
	return cfg
}

func TestSendEmail_NoRecipientSendsNothing(t *testing.T) {
	resend := useFakeResend(t, resendAccepts)
	smtpServer := startFakeSMTP(t, nil)

	err := SendEmail(withBothChannels(smtpServer.config()), EmailMessage{Subject: "Hello", HTML: "<p>x</p>"})

	assert.NoError(t, err)
	assert.Zero(t, resend.count())
	assert.Empty(t, smtpServer.delivered())
}

func TestSendEmail_NoChannelConfiguredIsNoOp(t *testing.T) {
	resend := useFakeResend(t, resendAccepts)

	// Credentials alone do not select a channel; the Use* flags do.
	err := SendEmail(config.Config{ResendAPIKey: "re_test_key", SMTPHost: "127.0.0.1"}, testEmail)

	assert.NoError(t, err)
	assert.Zero(t, resend.count())
}

func TestSendEmail_ResendOnly(t *testing.T) {
	resend := useFakeResend(t, resendAccepts)

	err := SendEmail(resendConfig(), testEmail)

	require.NoError(t, err)
	require.Equal(t, 1, resend.count())
	req := resend.requests[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "/emails", req.URL.Path)
	assert.Equal(t, "Bearer re_test_key", req.Header.Get("Authorization"))

	var payload struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		HTML    string   `json:"html"`
	}
	require.NoError(t, json.Unmarshal([]byte(resend.bodies[0]), &payload))
	assert.Equal(t, "crm@example.com", payload.From)
	assert.Equal(t, []string{"friend@example.com"}, payload.To)
	assert.Equal(t, "Hello", payload.Subject)
	assert.Equal(t, "<p>Hi there</p>", payload.HTML)
}

func TestSendEmail_ResendFailures(t *testing.T) {
	tests := []struct {
		name    string
		respond func() (*http.Response, error)
		wantErr string
	}{
		{"rejected by the API", resendRejects, "invalid from address"},
		{"rate limited", resendResponse(http.StatusTooManyRequests, `{"message":"slow down"}`), "slow down"}, //nolint:bodyclose // a factory; the SDK under test closes the body
		{"server error", resendResponse(http.StatusInternalServerError, `{"message":"boom"}`), "resend:"},    //nolint:bodyclose // a factory; the SDK under test closes the body
		{"network failure", func() (*http.Response, error) { return nil, errors.New("connection reset") }, "connection reset"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useFakeResend(t, tt.respond)

			err := SendEmail(resendConfig(), testEmail)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "resend:")
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Contains(t, err.Error(), "1 attempted")
		})
	}
}

func TestSendEmail_SMTPOnly(t *testing.T) {
	resend := useFakeResend(t, resendAccepts)
	server := startFakeSMTP(t, nil)

	err := SendEmail(server.config(), testEmail)

	require.NoError(t, err)
	assert.Zero(t, resend.count(), "Resend must not be used when only SMTP is selected")
	require.Len(t, server.delivered(), 1)
	authPlain, mailFrom, rcptTo := server.envelope()
	assert.Contains(t, mailFrom[0], "crm@example.com")
	assert.Contains(t, rcptTo[0], "friend@example.com")
	assert.Empty(t, authPlain, "no credentials configured, so no AUTH")

	message := server.delivered()[0]
	assert.Contains(t, message, "From: crm@example.com\r\n")
	assert.Contains(t, message, "To: friend@example.com\r\n")
	assert.Contains(t, message, "Subject: Hello\r\n")
	assert.Contains(t, message, "<p>Hi there</p>")
}

func TestSendEmail_SMTPAuthenticatesWithConfiguredCredentials(t *testing.T) {
	server := startFakeSMTP(t, nil)
	cfg := server.config()
	cfg.SMTPUsername = "smtp-user"
	cfg.SMTPPassword = "smtp-pass"

	require.NoError(t, SendEmail(cfg, testEmail))

	authPlain, _, _ := server.envelope()
	require.Len(t, authPlain, 1)
	assert.Equal(t, "\x00smtp-user\x00smtp-pass", authPlain[0])
	assert.Len(t, server.delivered(), 1)
}

func TestSendEmail_SMTPFailures(t *testing.T) {
	for _, verb := range []string{"AUTH", "MAIL", "RCPT", "DATA"} {
		t.Run("server rejects "+verb, func(t *testing.T) {
			server := startFakeSMTP(t, func(s *fakeSMTP) { s.rejectOn = verb })
			cfg := server.config()
			cfg.SMTPUsername = "smtp-user"
			cfg.SMTPPassword = "smtp-pass"

			err := SendEmail(cfg, testEmail)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "smtp:")
			assert.Empty(t, server.delivered())
		})
	}

	t.Run("server unreachable", func(t *testing.T) {
		err := SendEmail(closedSMTPConfig(t), testEmail)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "smtp:")
	})
}

func TestSendEmail_SMTPRefusesUntrustedCertificateOnSTARTTLS(t *testing.T) {
	selfSigned := httptest.NewTLSServer(nil)
	defer selfSigned.Close()

	server := startFakeSMTP(t, func(s *fakeSMTP) {
		s.startTLS = &tls.Config{Certificates: selfSigned.TLS.Certificates}
	})

	err := SendEmail(server.config(), testEmail)

	require.Error(t, err)
	assert.Empty(t, server.delivered(), "must not fall back to plaintext after a failed upgrade")
}

// The success path of implicit TLS cannot be tested here: sendSMTPImplicitTLS verifies against
// the system roots and offers no way to supply a test CA.
func TestSendEmail_ImplicitTLSFailures(t *testing.T) {
	t.Run("server does not speak TLS", func(t *testing.T) {
		server := startFakeSMTP(t, nil)
		cfg := server.config()
		cfg.SMTPUseTLS = true

		err := SendEmail(cfg, testEmail)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "tls dial")
		assert.Empty(t, server.delivered())
	})

	t.Run("untrusted certificate", func(t *testing.T) {
		selfSigned := httptest.NewTLSServer(nil)
		defer selfSigned.Close()
		addr := selfSigned.Listener.Addr().(*net.TCPAddr)

		err := SendEmail(config.Config{
			UseSMTP:       true,
			SMTPUseTLS:    true,
			SMTPHost:      addr.IP.String(),
			SMTPPort:      addr.Port,
			SMTPFromEmail: "crm@example.com",
		}, testEmail)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "tls dial")
	})

	t.Run("nothing listening", func(t *testing.T) {
		cfg := closedSMTPConfig(t)
		cfg.SMTPUseTLS = true

		err := SendEmail(cfg, testEmail)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "tls dial")
	})
}

func TestSendEmail_BothChannels(t *testing.T) {
	t.Run("both succeed and both are used", func(t *testing.T) {
		resend := useFakeResend(t, resendAccepts)
		server := startFakeSMTP(t, nil)

		require.NoError(t, SendEmail(withBothChannels(server.config()), testEmail))

		assert.Equal(t, 1, resend.count())
		assert.Len(t, server.delivered(), 1)
	})

	t.Run("Resend failing does not stop SMTP or fail the send", func(t *testing.T) {
		useFakeResend(t, resendRejects)
		server := startFakeSMTP(t, nil)

		require.NoError(t, SendEmail(withBothChannels(server.config()), testEmail))

		assert.Len(t, server.delivered(), 1)
	})

	t.Run("SMTP failing does not fail the send when Resend worked", func(t *testing.T) {
		resend := useFakeResend(t, resendAccepts)

		require.NoError(t, SendEmail(withBothChannels(closedSMTPConfig(t)), testEmail))

		assert.Equal(t, 1, resend.count())
	})

	t.Run("all failing reports each channel", func(t *testing.T) {
		useFakeResend(t, resendRejects)

		err := SendEmail(withBothChannels(closedSMTPConfig(t)), testEmail)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "2 attempted")
		assert.Contains(t, err.Error(), "resend:")
		assert.Contains(t, err.Error(), "smtp:")
	})
}

func TestSendEmail_RecipientCannotInjectSMTPCommands(t *testing.T) {
	server := startFakeSMTP(t, nil)

	err := SendEmail(server.config(), EmailMessage{
		To:      "friend@example.com\r\nRCPT TO:<victim@example.com>",
		Subject: "Hello",
		HTML:    "<p>x</p>",
	})

	require.Error(t, err)
	_, _, rcptTo := server.envelope()
	assert.Empty(t, rcptTo)
	assert.Empty(t, server.delivered())
}

func TestBuildSMTPMessage(t *testing.T) {
	t.Run("separates headers from the HTML body", func(t *testing.T) {
		message := string(buildSMTPMessage("crm@example.com", testEmail))

		headers, body, found := strings.Cut(message, "\r\n\r\n")
		require.True(t, found)
		assert.Equal(t, "<p>Hi there</p>", body)
		assert.Contains(t, headers, "MIME-Version: 1.0")
		assert.Contains(t, headers, `Content-Type: text/html; charset="UTF-8"`)
	})

	t.Run("encodes a non-ASCII subject", func(t *testing.T) {
		message := string(buildSMTPMessage("crm@example.com", EmailMessage{To: "a@example.com", Subject: "Grüße", HTML: "x"}))

		headers, _, _ := strings.Cut(message, "\r\n\r\n")
		assert.NotContains(t, headers, "Grüße")

		var subject string
		for _, line := range strings.Split(headers, "\r\n") {
			if value, ok := strings.CutPrefix(line, "Subject: "); ok {
				subject = value
			}
		}
		decoded, err := new(mime.WordDecoder).DecodeHeader(subject)
		require.NoError(t, err)
		assert.Equal(t, "Grüße", decoded)
	})

	t.Run("a newline in the subject cannot add a header", func(t *testing.T) {
		message := string(buildSMTPMessage("crm@example.com", EmailMessage{
			To:      "a@example.com",
			Subject: "Hello\r\nBcc: victim@example.com",
			HTML:    "x",
		}))

		headers, _, _ := strings.Cut(message, "\r\n\r\n")
		for _, line := range strings.Split(headers, "\r\n") {
			assert.False(t, strings.HasPrefix(line, "Bcc:"), "injected header in %q", headers)
		}
	})
}

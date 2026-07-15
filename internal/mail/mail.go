// Package mail delivers moth's transactional emails. Two transports: SMTP
// for production and a console logger so the whole auth flow works in dev
// with zero setup.
package mail

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"strings"
)

// Message is one email ready to send.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// Mailer sends transactional emails.
type Mailer interface {
	Send(ctx context.Context, m Message) error
}

// Console logs the full email instead of sending it — the dev default.
type Console struct {
	Log *slog.Logger
}

func (c Console) Send(_ context.Context, m Message) error {
	log := c.Log
	if log == nil {
		log = slog.Default()
	}
	log.Info("email (console transport)\n" +
		"To: " + m.To + "\n" +
		"Subject: " + m.Subject + "\n\n" +
		m.Text)
	return nil
}

// SMTP sends through a configured SMTP relay, with STARTTLS when the
// server offers it and PLAIN auth when a username is configured.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (s SMTP) Send(_ context.Context, m Message) error {
	var auth smtp.Auth
	if s.Username != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}
	body, err := buildMIME(s.From, m)
	if err != nil {
		return fmt.Errorf("build email: %w", err)
	}
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	if err := smtp.SendMail(addr, auth, s.From, []string{m.To}, body); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

// buildMIME assembles a multipart/alternative message with plain-text and
// HTML parts.
func buildMIME(from string, m Message) ([]byte, error) {
	var buf bytes.Buffer
	mp := multipart.NewWriter(&buf)

	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", m.To)
	fmt.Fprintf(&buf, "Subject: %s\r\n", sanitizeHeader(m.Subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", mp.Boundary())

	for _, part := range []struct{ contentType, body string }{
		{"text/plain; charset=utf-8", m.Text},
		{"text/html; charset=utf-8", m.HTML},
	} {
		if part.body == "" {
			continue
		}
		w, err := mp.CreatePart(map[string][]string{
			"Content-Type":              {part.contentType},
			"Content-Transfer-Encoding": {"quoted-printable"},
		})
		if err != nil {
			return nil, err
		}
		qp := quotedprintable.NewWriter(w)
		if _, err := qp.Write([]byte(part.body)); err != nil {
			return nil, err
		}
		if err := qp.Close(); err != nil {
			return nil, err
		}
	}
	if err := mp.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// sanitizeHeader strips CR/LF so message fields can never inject headers.
func sanitizeHeader(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
}

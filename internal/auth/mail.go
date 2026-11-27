package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// LogMailer writes the verification link to the log instead of sending it.
// It is used when no SMTP server is configured.
type LogMailer struct{}

func (LogMailer) SendVerification(ctx context.Context, email, link string) error {
	slog.InfoContext(ctx, "verification email (SMTP not configured)", "to", email, "link", link)
	return nil
}

// SMTPMailer sends emails through an SMTP server. Username and Password are
// optional; without them the server must accept mail without authentication
// (as Mailpit does in the local Docker Compose setup).
type SMTPMailer struct {
	Addr     string // host:port
	From     string
	Username string
	Password string
}

func (m SMTPMailer) SendVerification(_ context.Context, email, link string) error {
	body := strings.Join([]string{
		"Вітаємо в Тамагочі!",
		"",
		"Щоб підтвердити адресу електронної пошти, відкрийте посилання:",
		link,
		"",
		"Посилання дійсне 48 годин. Якщо ви не реєструвалися, просто проігноруйте цей лист.",
	}, "\r\n")
	return m.send(email, "Підтвердження адреси електронної пошти", body)
}

func (m SMTPMailer) send(to, subject, body string) error {
	if strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("invalid recipient %q", to)
	}
	// The envelope sender must be a bare address; the header keeps the name.
	from, err := mail.ParseAddress(m.From)
	if err != nil {
		return fmt.Errorf("SMTP_FROM: %w", err)
	}
	msg := strings.Join([]string{
		"From: " + from.String(),
		"To: " + to,
		"Subject: =?UTF-8?B?" + base64Std(subject) + "?=",
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")
	var auth smtp.Auth
	if m.Username != "" {
		host, _, err := net.SplitHostPort(m.Addr)
		if err != nil {
			return fmt.Errorf("smtp address: %w", err)
		}
		auth = smtp.PlainAuth("", m.Username, m.Password, host)
	}
	if err := smtp.SendMail(m.Addr, auth, from.Address, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("send mail to %s: %w", to, err)
	}
	return nil
}

func base64Std(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

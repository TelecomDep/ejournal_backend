package app

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
)

type Mailer struct {
	host     string
	port     string
	user     string
	password string
	from     string
	baseURL  string
}

func NewMailer(host, port, user, password, from, baseURL string) *Mailer {
	return &Mailer{
		host:     strings.TrimSpace(host),
		port:     strings.TrimSpace(port),
		user:     strings.TrimSpace(user),
		password: password,
		from:     strings.TrimSpace(from),
		baseURL:  strings.TrimRight(strings.TrimSpace(baseURL), "/"),
	}
}

func (m *Mailer) SendPasswordReset(to, token string) error {
	if m.host == "" {
		log.Printf("SMTP_HOST is not configured; skipping sending password reset email to %s (token: %s)", to, token)
		return nil
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", m.baseURL, token)
	subject := "Восстановление пароля / Password Reset"
	body := fmt.Sprintf("Здравствуйте!\n\nДля сброса пароля перейдите по следующей ссылке:\n%s\n\nСсылка действительна в течение 15 минут.\n\nЕсли вы не запрашивали сброс пароля, проигнорируйте это письмо.\n\n---\n\nHello!\n\nTo reset your password, click the following link:\n%s\n\nThis link is valid for 15 minutes.\n\nIf you did not request this, please ignore this email.", resetLink, resetLink)

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", to, m.from, subject, body))

	addr := net.JoinHostPort(m.host, m.port)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("dial smtp server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("new smtp client: %w", err)
	}
	defer client.Close()

	// If using port 587 (Submission) or 465, STARTTLS is standard
	if m.port == "587" || m.port == "25" {
		tlsConfig := &tls.Config{
			ServerName:         m.host,
			InsecureSkipVerify: true, // Allowed for Docker internal networks or dynamic certificates
		}
		// Try STARTTLS, ignore if not supported by the server (e.g. port 25 without TLS support inside docker net)
		_ = client.StartTLS(tlsConfig)
	}

	if m.user != "" && m.password != "" {
		auth := smtp.PlainAuth("", m.user, m.password, m.host)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err = client.Mail(m.from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}

	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data command: %w", err)
	}

	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	if err = w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}

	return client.Quit()
}

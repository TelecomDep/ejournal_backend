package app

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type Mailer struct {
	host          string
	port          string
	user          string
	password      string
	from          string
	baseURL       string
	tlsServerName string
	caFile        string
}

func NewMailer(host, port, user, password, from, baseURL, tlsServerName, caFile string) *Mailer {
	return &Mailer{
		host:          strings.TrimSpace(host),
		port:          strings.TrimSpace(port),
		user:          strings.TrimSpace(user),
		password:      password,
		from:          strings.TrimSpace(from),
		baseURL:       strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		tlsServerName: strings.TrimSpace(tlsServerName),
		caFile:        strings.TrimSpace(caFile),
	}
}

func (m *Mailer) Available() bool {
	return m != nil && m.host != "" && m.port != "" && m.from != ""
}

func validMailAddress(value string) (*mail.Address, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return nil, errors.New("invalid mail address")
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address == "" || strings.ContainsAny(parsed.Address, "\r\n") {
		return nil, errors.New("invalid mail address")
	}
	return parsed, nil
}

func (m *Mailer) smtpClient() (*smtp.Client, error) {
	if !m.Available() {
		return nil, errors.New("SMTP is not configured")
	}
	addr := net.JoinHostPort(m.host, m.port)
	serverName := m.tlsServerName
	if serverName == "" {
		serverName = m.host
	}
	tlsConfig := &tls.Config{ServerName: serverName, MinVersion: tls.VersionTLS12}
	if m.caFile != "" {
		caPEM, err := os.ReadFile(m.caFile)
		if err != nil {
			return nil, fmt.Errorf("read SMTP CA: %w", err)
		}
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("SMTP CA file contains no certificates")
		}
		tlsConfig.RootCAs = roots
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	var (
		client *smtp.Client
		err    error
	)
	if m.port == "465" {
		conn, dialErr := tls.DialWithDialer(dialer, "tcp", addr, tlsConfig)
		if dialErr != nil {
			return nil, fmt.Errorf("dial SMTP TLS: %w", dialErr)
		}
		_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
		client, err = smtp.NewClient(conn, m.host)
	} else {
		conn, dialErr := dialer.Dial("tcp", addr)
		if dialErr != nil {
			return nil, fmt.Errorf("dial SMTP: %w", dialErr)
		}
		_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
		client, err = smtp.NewClient(conn, m.host)
		if err == nil {
			if supported, _ := client.Extension("STARTTLS"); !supported {
				_ = client.Close()
				return nil, errors.New("SMTP server does not support STARTTLS")
			}
			err = client.StartTLS(tlsConfig)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("create secure SMTP client: %w", err)
	}
	if (m.user == "") != (m.password == "") {
		_ = client.Close()
		return nil, errors.New("SMTP credentials are incomplete")
	}
	if m.user != "" {
		if err := client.Auth(smtp.PlainAuth("", m.user, m.password, m.host)); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("SMTP authentication: %w", err)
		}
	}
	return client, nil
}

func (m *Mailer) send(to, subject, body string) error {
	recipient, err := validMailAddress(to)
	if err != nil {
		return err
	}
	sender, err := validMailAddress(m.from)
	if err != nil {
		return err
	}
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		return fmt.Errorf("generate message id: %w", err)
	}
	messageID := fmt.Sprintf("<%x@%s>", randBytes, m.host)
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	message := []byte(fmt.Sprintf(
		"To: %s\r\nFrom: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		recipient.String(), sender.String(), encodedSubject, time.Now().Format(time.RFC1123Z), messageID, body,
	))

	client, err := m.smtpClient()
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.Mail(sender.Address); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}
	if err := client.Rcpt(recipient.Address); err != nil {
		return fmt.Errorf("SMTP RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := w.Write(message); err != nil {
		_ = w.Close()
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close SMTP message: %w", err)
	}
	return client.Quit()
}

func (m *Mailer) SendPasswordReset(to, token string) error {
	resetLink := fmt.Sprintf("%s/#/reset-password?token=%s", m.baseURL, token)
	body := fmt.Sprintf("Здравствуйте!\n\nДля сброса пароля перейдите по ссылке:\n%s\n\nСсылка действительна 15 минут. Если вы не запрашивали сброс, проигнорируйте письмо.\n\n---\n\nHello!\n\nReset your password using this link:\n%s\n\nThe link is valid for 15 minutes. If you did not request this, ignore this email.", resetLink, resetLink)
	return m.send(to, "Восстановление пароля / Password Reset", body)
}

func (m *Mailer) SendEmailConfirmation(to, code string) error {
	body := fmt.Sprintf("Здравствуйте!\n\nКод подтверждения email: %s\n\nНикому не сообщайте этот код.\n\n---\n\nHello!\n\nEmail confirmation code: %s\n\nDo not share this code.", code, code)
	return m.send(to, "Подтверждение Email / Email Confirmation", body)
}

func (m *Mailer) Send2FAEnableConfirmation(to, code string) error {
	body := fmt.Sprintf("Здравствуйте!\n\nКод для подключения 2FA: %s\n\nКод действует 15 минут. Никому его не сообщайте.\n\n---\n\nHello!\n\n2FA setup code: %s\n\nThe code is valid for 15 minutes. Do not share it.", code, code)
	return m.send(to, "Подтверждение включения 2FA / Enable 2FA Confirmation", body)
}

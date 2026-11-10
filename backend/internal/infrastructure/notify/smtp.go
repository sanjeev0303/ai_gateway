// Package notify sends platform alert emails over SMTP.
package notify

import (
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// SMTPMailer sends plain-text mail. Without a username it sends without
// authentication, which is what a local catcher such as Mailpit expects.
// net/smtp upgrades to STARTTLS whenever the server offers it.
type SMTPMailer struct {
	addr     string
	host     string
	username string
	password string
	from     string
}

func NewSMTPMailer(host string, port int, username, password, from string) *SMTPMailer {
	return &SMTPMailer{
		addr:     net.JoinHostPort(host, strconv.Itoa(port)),
		host:     host,
		username: username,
		password: password,
		from:     from,
	}
}

func (m *SMTPMailer) Send(to []string, subject, body string) error {
	if len(to) == 0 {
		return nil
	}
	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	var msg strings.Builder
	msg.WriteString("From: " + headerSafe(m.from) + "\r\n")
	msg.WriteString("To: " + headerSafe(strings.Join(to, ", ")) + "\r\n")
	msg.WriteString("Subject: " + headerSafe(subject) + "\r\n")
	msg.WriteString("Date: " + time.Now().UTC().Format(time.RFC1123Z) + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n")
	msg.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))

	if err := smtp.SendMail(m.addr, auth, m.from, to, []byte(msg.String())); err != nil {
		return fmt.Errorf("send alert email: %w", err)
	}
	return nil
}

// headerSafe strips line breaks so text placed in a header cannot add headers.
func headerSafe(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
}

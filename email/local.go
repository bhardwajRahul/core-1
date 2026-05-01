package email

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/staticbackendhq/core/config"
)

type Local struct{}

func (Local) Send(data SendMailData) error {
	// Validate email addresses
	if len(data.To) == 0 || !strings.Contains(data.To, "@") {
		return fmt.Errorf("empty To email")
	}
	if len(data.From) == 0 || !strings.Contains(data.From, "@") {
		return fmt.Errorf("empty From email")
	}

	rawEmail, err := BuildRawEmail(data)
	if err != nil {
		return fmt.Errorf("failed to build raw email: %w", err)
	}

	// Send email via SMTP
	// Mailpit doesn't require authentication
	addr := config.Current.MailpitSMTPAddr
	if addr == "" {
		addr = "localhost:1025"
	}

	err = smtp.SendMail(
		addr,
		nil, // no auth
		data.From,
		[]string{data.To},
		rawEmail,
	)
	if err != nil {
		return fmt.Errorf("failed to send email via Mailpit: %w", err)
	}

	return nil
}

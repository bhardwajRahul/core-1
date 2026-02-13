package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"strings"
	"time"
)

const (
	MailProviderDev = "dev"
	MailProviderSES = "ses"
)

// Attachment represents an email attachment
type Attachment struct {
	URL         string `json:"url"`         // URL where the attachment was fetched from
	Body        []byte `json:"body"`        // Raw bytes of the attachment
	ContentType string `json:"contentType"` // MIME type of the attachment
	Filename    string `json:"filename"`    // Name of the file
}

// SendMailData contains necessary fields to send an email
type SendMailData struct {
	From     string `json:"from"`
	FromName string `json:"fromName"`
	To       string `json:"to"`
	ToName   string `json:"toName"`
	Subject  string `json:"subject"`
	HTMLBody string `json:"htmlBody"`
	TextBody string `json:"textBody"`
	ReplyTo  string `json:"replyTo"`

	Body string `json:"body"`

	Attachments     []Attachment `json:"attachments"`
	IsTransactional bool         `json:"isTransactional"`
}

// Mailer is used to have different implementation for sending email
type Mailer interface {
	// Send sends the email
	Send(SendMailData) error
}

// BuildRawEmail constructs a raw MIME email message from SendMailData.
// This is shared across all email implementations (Dev, SES, etc.) to ensure
// consistent email formatting and support for attachments and transactional flags.
func BuildRawEmail(data SendMailData) ([]byte, error) {
	var buf bytes.Buffer

	// Build From header with optional name
	from := data.From
	if data.FromName != "" {
		from = fmt.Sprintf("%s <%s>", data.FromName, data.From)
	}

	// Build To header with optional name
	to := data.To
	if data.ToName != "" {
		to = fmt.Sprintf("%s <%s>", data.ToName, data.To)
	}

	// Set ReplyTo to From if not specified
	replyTo := data.ReplyTo
	if replyTo == "" {
		replyTo = data.From
	}

	// Write email headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", replyTo))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", data.Subject))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	buf.WriteString("MIME-Version: 1.0\r\n")

	// Add transactional email header if flagged
	if data.IsTransactional {
		buf.WriteString("X-Transactional: true\r\n")
		buf.WriteString("Precedence: bulk\r\n")
	}

	// Create multipart writer
	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", boundary))

	// Add text/html alternative part
	if data.HTMLBody != "" || data.TextBody != "" {
		// Create alternative part for text and HTML
		altWriter := multipart.NewWriter(&buf)
		altBoundary := altWriter.Boundary()

		partHeader := textproto.MIMEHeader{}
		partHeader.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=\"%s\"", altBoundary))
		_, err := writer.CreatePart(partHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to create alternative part: %w", err)
		}

		// Add text part
		if data.TextBody != "" {
			textHeader := textproto.MIMEHeader{}
			textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
			textHeader.Set("Content-Transfer-Encoding", "quoted-printable")
			textPart, err := altWriter.CreatePart(textHeader)
			if err != nil {
				return nil, fmt.Errorf("failed to create text part: %w", err)
			}
			textPart.Write([]byte(data.TextBody))
		}

		// Add HTML part
		if data.HTMLBody != "" {
			htmlHeader := textproto.MIMEHeader{}
			htmlHeader.Set("Content-Type", "text/html; charset=UTF-8")
			htmlHeader.Set("Content-Transfer-Encoding", "quoted-printable")
			htmlPart, err := altWriter.CreatePart(htmlHeader)
			if err != nil {
				return nil, fmt.Errorf("failed to create HTML part: %w", err)
			}
			htmlPart.Write([]byte(data.HTMLBody))
		}

		altWriter.Close()
	}

	// Add attachments
	for _, attachment := range data.Attachments {
		if len(attachment.Body) == 0 {
			continue
		}

		contentType := attachment.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		filename := attachment.Filename
		if filename == "" {
			filename = "attachment"
		}

		attachHeader := textproto.MIMEHeader{}
		attachHeader.Set("Content-Type", contentType)
		attachHeader.Set("Content-Transfer-Encoding", "base64")
		attachHeader.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		attachPart, err := writer.CreatePart(attachHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to create attachment part: %w", err)
		}

		// Encode attachment body as base64
		encoded := base64.StdEncoding.EncodeToString(attachment.Body)
		// Write base64 in 76-character lines per RFC 2045
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			attachPart.Write([]byte(encoded[i:end] + "\r\n"))
		}
	}

	writer.Close()

	return buf.Bytes(), nil
}

// ExtractEmailAddress extracts the email address from a formatted string like "Name <email@example.com>"
// or returns the input if it's already just an email address
func ExtractEmailAddress(email string) string {
	email = strings.TrimSpace(email)
	if strings.Contains(email, "<") && strings.Contains(email, ">") {
		start := strings.Index(email, "<")
		end := strings.Index(email, ">")
		if start < end {
			return email[start+1 : end]
		}
	}
	return email
}

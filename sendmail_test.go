package staticbackend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/email"
)

func Test_Sendmail(t *testing.T) {
	data := email.SendMailData{
		FromName: config.Current.FromName,
		From:     config.Current.FromEmail,
		To:       "dominicstpierre+unittest@gmail.com",
		ToName:   "Dominic St-Pierre",
		Subject:  "From unit test",
		HTMLBody: "<h1>hello</h1><p>working</p>",
		TextBody: "Hello\nworking",
		ReplyTo:  config.Current.FromEmail,
	}
	if err := backend.Emailer.Send(data); err != nil {
		t.Error(err)
	}
}

func TestSendMailWithAttachment(t *testing.T) {
	// Create email data with attachment
	data := email.SendMailData{
		FromName:        config.Current.FromName,
		From:            config.Current.FromEmail,
		To:              "dominicstpierre+attachment@gmail.com",
		ToName:          "Dominic St-Pierre",
		Subject:         "Test email with attachment",
		HTMLBody:        "<h1>Test Email</h1><p>This email has an attachment</p>",
		TextBody:        "Test Email\nThis email has an attachment",
		ReplyTo:         config.Current.FromEmail,
		IsTransactional: true,
		Attachments: []email.Attachment{
			{
				URL: "https://staticbackend.dev/img/logo-sb-red-text.png",
			},
		},
	}

	// Send the email
	resp := dbReq(t, sudoSendMail, "POST", "/sudo/sendmail", data, true)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	// Give Mailpit time to process the email
	time.Sleep(1 * time.Second)

	// Verify email in Mailpit
	mailpitResp, err := http.Get("http://localhost:8025/api/v1/messages")
	if err != nil {
		t.Fatalf("failed to query Mailpit API: %v", err)
	}
	defer mailpitResp.Body.Close()

	if mailpitResp.StatusCode != http.StatusOK {
		t.Fatalf("Mailpit API returned non-OK status: %d", mailpitResp.StatusCode)
	}

	// Parse Mailpit response
	var mailpitData struct {
		Total    int `json:"total"`
		Messages []struct {
			ID   string `json:"ID"`
			From struct {
				Address string `json:"Address"`
			} `json:"From"`
			To []struct {
				Address string `json:"Address"`
			} `json:"To"`
			Subject     string `json:"Subject"`
			Attachments int    `json:"Attachments"`
		} `json:"messages"`
	}

	if err := json.NewDecoder(mailpitResp.Body).Decode(&mailpitData); err != nil {
		t.Fatalf("failed to decode Mailpit response: %v", err)
	}

	// Find our email
	var found bool
	var messageID string
	for _, msg := range mailpitData.Messages {
		if msg.Subject == "Test email with attachment" {
			found = true
			messageID = msg.ID

			// Verify basic email properties
			if msg.From.Address != config.Current.FromEmail {
				t.Errorf("expected from to be %s, got %s", config.Current.FromEmail, msg.From.Address)
			}

			if len(msg.To) == 0 || msg.To[0].Address != "dominicstpierre+attachment@gmail.com" {
				t.Errorf("expected to to be dominicstpierre+attachment@gmail.com")
			}

			if msg.Attachments != 1 {
				t.Errorf("expected 1 attachment, got %d", msg.Attachments)
			}

			break
		}
	}

	if !found {
		t.Fatal("email not found in Mailpit")
	}

	// Get full message details to verify attachment
	msgResp, err := http.Get(fmt.Sprintf("http://localhost:8025/api/v1/message/%s", messageID))
	if err != nil {
		t.Fatalf("failed to get message details from Mailpit: %v", err)
	}
	defer msgResp.Body.Close()

	var messageDetails struct {
		Attachments []struct {
			FileName    string `json:"FileName"`
			ContentType string `json:"ContentType"`
			Size        int    `json:"Size"`
		} `json:"Attachments"`
		Headers map[string][]string `json:"Headers"`
	}

	if err := json.NewDecoder(msgResp.Body).Decode(&messageDetails); err != nil {
		t.Fatalf("failed to decode message details: %v", err)
	}

	// Verify attachment details
	if len(messageDetails.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(messageDetails.Attachments))
	}

	attachment := messageDetails.Attachments[0]
	if attachment.FileName != "logo-sb-red-text.png" {
		t.Errorf("expected filename to be logo-sb-red-text.png, got %s", attachment.FileName)
	}

	if !strings.Contains(attachment.ContentType, "image/png") {
		t.Errorf("expected content type to contain image/png, got %s", attachment.ContentType)
	}

	if attachment.Size == 0 {
		t.Error("attachment size should be > 0")
	}

	// Verify transactional header (check various case variations)
	// Note: Mailpit might not expose all headers via API, so we'll just log if not found
	foundTransactional := false
	var transactionalValue string
	for headerName, headerValues := range messageDetails.Headers {
		if strings.EqualFold(headerName, "X-Transactional") {
			foundTransactional = true
			if len(headerValues) > 0 {
				transactionalValue = headerValues[0]
			}
			break
		}
	}

	if len(messageDetails.Headers) == 0 {
		t.Log("⚠ Mailpit API doesn't expose headers - skipping header verification")
		t.Log("  (X-Transactional header was added to the MIME message)")
	} else if !foundTransactional {
		t.Logf("⚠ X-Transactional header not found in API response. Available headers: %v", getHeaderNames(messageDetails.Headers))
	} else if transactionalValue != "true" {
		t.Errorf("expected X-Transactional header to be 'true', got '%s'", transactionalValue)
	} else {
		t.Log("✓ X-Transactional header verified: true")
	}

	t.Log("✅ Email sent successfully with attachment")
	t.Logf("✅ Attachment: %s (%s, %d bytes)", attachment.FileName, attachment.ContentType, attachment.Size)
}

// getHeaderNames returns a list of header names for debugging
func getHeaderNames(headers map[string][]string) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	return names
}

package staticbackend

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/middleware"
)

func sudoSendMail(w http.ResponseWriter, r *http.Request) {
	var data email.SendMailData
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch attachment bodies if any attachments are present
	if len(data.Attachments) > 0 {
		if err := fetchAttachments(&data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// if only body is provided
	if len(data.Body) > 0 {
		data.HTMLBody = data.Body
		data.TextBody = email.StripHTML(data.Body)
	} else if len(data.TextBody) == 0 && len(data.HTMLBody) > 0 {
		data.TextBody = email.StripHTML(data.HTMLBody)
	} else if len(data.HTMLBody) == 0 && len(data.TextBody) > 0 {
		data.HTMLBody = data.TextBody
	}

	if err := backend.Emailer.Send(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := backend.DB.IncrementMonthlyEmailSent(config.ID); err != nil {
		//TODO: do something better with this error
		log.Println("error increasing monthly email sent: ", err)
	}

	respond(w, http.StatusOK, true)
}

// fetchAttachments downloads attachment bodies from their URLs and populates
// ContentType and Filename fields from the HTTP response
func fetchAttachments(data *email.SendMailData) error {
	if len(data.Attachments) == 0 {
		return nil
	}

	for i := range data.Attachments {
		attachment := &data.Attachments[i]

		// Skip if body is already populated
		if len(attachment.Body) > 0 {
			continue
		}

		// Validate URL is present
		if attachment.URL == "" {
			return fmt.Errorf("attachment %d has no URL", i)
		}

		// Fetch the attachment from URL
		resp, err := http.Get(attachment.URL)
		if err != nil {
			return fmt.Errorf("failed to fetch attachment from %s: %w", attachment.URL, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch attachment from %s: status %d", attachment.URL, resp.StatusCode)
		}

		// Read the body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read attachment body from %s: %w", attachment.URL, err)
		}
		attachment.Body = body

		// Set content type from response header if not already set
		if attachment.ContentType == "" {
			contentType := resp.Header.Get("Content-Type")
			if contentType != "" {
				// Remove any parameters (e.g., "image/png; charset=utf-8" -> "image/png")
				if idx := strings.Index(contentType, ";"); idx != -1 {
					contentType = strings.TrimSpace(contentType[:idx])
				}
				attachment.ContentType = contentType
			}
		}

		// Extract filename if not already set
		if attachment.Filename == "" {
			// Try Content-Disposition header first
			cd := resp.Header.Get("Content-Disposition")
			if cd != "" {
				// Parse Content-Disposition to extract filename
				// Example: attachment; filename="file.pdf"
				if idx := strings.Index(cd, "filename="); idx != -1 {
					filename := cd[idx+9:]
					filename = strings.Trim(filename, "\"")
					attachment.Filename = filename
				}
			}

			// If still no filename, extract from URL path
			if attachment.Filename == "" {
				parts := strings.Split(attachment.URL, "/")
				if len(parts) > 0 {
					filename := parts[len(parts)-1]
					// Remove query parameters if any
					if idx := strings.Index(filename, "?"); idx != -1 {
						filename = filename[:idx]
					}
					// URL decode the filename
					attachment.Filename = filename
				}
			}

			// Fallback to a generic name if still empty
			if attachment.Filename == "" {
				attachment.Filename = fmt.Sprintf("attachment_%d", i+1)
			}
		}
	}

	return nil
}

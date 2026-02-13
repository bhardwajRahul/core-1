package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/config"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

type AWSSES struct{}

func (AWSSES) Send(data SendMailData) error {
	if len(data.To) == 0 || !strings.Contains(data.To, "@") {
		return fmt.Errorf("empty To email")
	}

	if len(data.ReplyTo) == 0 {
		data.ReplyTo = data.From
	}

	region := strings.TrimSpace(config.Current.S3Region)
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		return err
	}

	// Create an SES client.
	svc := ses.NewFromConfig(cfg)

	rawEmail, err := BuildRawEmail(data)
	if err != nil {
		return fmt.Errorf("failed to build raw email: %w", err)
	}

	// Send raw email (works for both emails with and without attachments)
	input := &ses.SendRawEmailInput{
		RawMessage: &types.RawMessage{
			Data: rawEmail,
		},
	}

	if _, err := svc.SendRawEmail(context.TODO(), input); err != nil {
		return fmt.Errorf("failed to send email via SES: %w", err)
	}

	return nil
}

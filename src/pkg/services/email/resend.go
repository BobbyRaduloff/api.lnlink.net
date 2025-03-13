package email

import (
	"context"
	"fmt"
	"time"

	"api.lnlink.net/src/pkg/errs"
	"api.lnlink.net/src/pkg/global"

	"github.com/resend/resend-go/v2"
)

func SendEmail(recipient string, subject string, html string, text string) error {
	client := resend.NewClient(global.RESEND_API_KEY)

	params := &resend.SendEmailRequest{
		From:    global.RESEND_FROM,
		To:      []string{recipient},
		Subject: subject,
		Html:    html,
		Text:    text,
	}

	resp, err := client.Emails.SendWithContext(context.TODO(), params)
	if err != nil {
		return err
	}

	// Intervals at which we will check for the email delivery status
	// after 1, 5, 10, 15, 30, 45, and 60 seconds
	intervals := []time.Duration{1, 4, 5, 5, 15, 15, 15}

	for _, sec := range intervals {
		time.Sleep(sec * time.Second)
		resp2, err := client.Emails.Get(resp.Id)
		errs.Invariant(err == nil, "failed to get email status from resend")
		if resp2.LastEvent == "delivered" {
			return nil
		}
	}

	return fmt.Errorf("failed to deliver email within 1 minute timeout")
}

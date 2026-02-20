package notification

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SMSProvider defines the interface for sending SMS messages.
// Implement this interface to swap providers (Twilio, Nexmo, etc.).
type SMSProvider interface {
	Send(ctx context.Context, to, from, message string) error
	Name() string
}

// ---------------------------------------------------------------------------
// LogProvider — logs SMS to stdout (dev / testing)
// ---------------------------------------------------------------------------

type LogProvider struct{}

func (p *LogProvider) Send(_ context.Context, to, from, message string) error {
	log.Printf("[SMS-LOG] to=%s from=%s message=%q", to, from, message)
	return nil
}

func (p *LogProvider) Name() string { return "log" }

// ---------------------------------------------------------------------------
// TwilioProvider — sends SMS via Twilio REST API (skeleton, production-ready shape)
// ---------------------------------------------------------------------------

type TwilioProvider struct {
	AccountSID string
	AuthToken  string
	client     *http.Client
}

func NewTwilioProvider(accountSID, authToken string) *TwilioProvider {
	return &TwilioProvider{
		AccountSID: accountSID,
		AuthToken:  authToken,
		client:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *TwilioProvider) Send(ctx context.Context, to, from, message string) error {
	apiURL := fmt.Sprintf(
		"https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json",
		p.AccountSID,
	)

	data := url.Values{}
	data.Set("To", to)
	data.Set("From", from)
	data.Set("Body", message)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("twilio: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(p.AccountSID, p.AuthToken)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("twilio: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("twilio: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (p *TwilioProvider) Name() string { return "twilio" }

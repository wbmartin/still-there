package sms

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const telnyxMessagesURL = "https://api.telnyx.com/v2/messages"

// TelnyxSender sends through Telnyx's Messaging API (POST /v2/messages).
type TelnyxSender struct {
	APIKey             string
	FromNumber         string
	MessagingProfileID string
	// URL is the messages endpoint (tests point it at a stub); "" is Telnyx.
	URL  string
	HTTP *http.Client
}

func (s TelnyxSender) Send(ctx context.Context, toE164 string, body string) (string, string, error) {
	payload, err := json.Marshal(map[string]string{
		"from":                 s.FromNumber,
		"to":                   toE164,
		"text":                 body,
		"messaging_profile_id": s.MessagingProfileID,
	})
	if err != nil {
		return "", "", err
	}
	url := s.URL
	if url == "" {
		url = telnyxMessagesURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := s.HTTP
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("telnyx: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode/100 != 2 {
		return "", "", fmt.Errorf("telnyx: HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	var out struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", "", fmt.Errorf("telnyx: reading response: %w", err)
	}
	return out.Data.ID, "sent", nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// webhookTolerance is how old a signed webhook may be, against replay.
const webhookTolerance = 5 * time.Minute

var ErrWebhookSignature = errors.New("telnyx webhook signature invalid")

// VerifyTelnyxWebhook checks a webhook's telnyx-signature-ed25519 header:
// an Ed25519 signature over "<telnyx-timestamp>|<raw body>" by the
// account's public key (base64, from the Telnyx portal), and a timestamp
// within webhookTolerance of now.
func VerifyTelnyxWebhook(publicKeyB64 string, signatureB64 string, timestamp string, body []byte, now time.Time) error {
	key, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return fmt.Errorf("telnyx public key: not a base64 Ed25519 key")
	}
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return ErrWebhookSignature
	}
	sec, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrWebhookSignature
	}
	if age := now.Sub(time.Unix(sec, 0)); age > webhookTolerance || age < -webhookTolerance {
		return ErrWebhookSignature
	}
	signed := append([]byte(timestamp+"|"), body...)
	if !ed25519.Verify(key, signed, sig) {
		return ErrWebhookSignature
	}
	return nil
}

// TelnyxInbound is the part of a message.received webhook we use.
type TelnyxInbound struct {
	EventType string
	From      string
	Text      string
}

func ParseTelnyxWebhook(body []byte) (TelnyxInbound, error) {
	var w struct {
		Data struct {
			EventType string `json:"event_type"`
			Payload   struct {
				Text string `json:"text"`
				From struct {
					PhoneNumber string `json:"phone_number"`
				} `json:"from"`
			} `json:"payload"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &w); err != nil {
		return TelnyxInbound{}, err
	}
	return TelnyxInbound{EventType: w.Data.EventType, From: w.Data.Payload.From.PhoneNumber, Text: w.Data.Payload.Text}, nil
}

package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Copied from src/api-srv/internal/sms/telnyx.go (VerifyTelnyxWebhook,
// ParseTelnyxWebhook): this module is split to its own repository later,
// so it imports nothing from the API.

// webhookTolerance is how old a signed webhook may be, against replay.
const webhookTolerance = 5 * time.Minute

var errWebhookSignature = errors.New("telnyx webhook signature invalid")

func parsePublicKey(publicKeyB64 string) (ed25519.PublicKey, error) {
	key, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf(
			"TELNYX_PUBLIC_KEY: not a base64 Ed25519 key",
		)
	}
	return key, nil
}

// verifyTelnyxWebhook checks a webhook's telnyx-signature-ed25519 header:
// an Ed25519 signature over "<telnyx-timestamp>|<raw body>" by the
// account's public key, and a timestamp within webhookTolerance of now.
func verifyTelnyxWebhook(
	key ed25519.PublicKey, signatureB64 string, timestamp string,
	body []byte, now time.Time,
) error {
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return errWebhookSignature
	}
	sec, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errWebhookSignature
	}
	age := now.Sub(time.Unix(sec, 0))
	if age > webhookTolerance || age < -webhookTolerance {
		return errWebhookSignature
	}
	signed := append([]byte(timestamp+"|"), body...)
	if !ed25519.Verify(key, signed, sig) {
		return errWebhookSignature
	}
	return nil
}

// telnyxInbound is the part of a webhook the relay stores.
type telnyxInbound struct {
	EventType string
	From      string
	To        string
	Text      string
}

func parseTelnyxWebhook(body []byte) (telnyxInbound, error) {
	type phone struct {
		PhoneNumber string `json:"phone_number"`
	}
	var w struct {
		Data struct {
			EventType string `json:"event_type"`
			Payload   struct {
				Text   string  `json:"text"`
				From   phone   `json:"from"`
				ToList []phone `json:"to"`
			} `json:"payload"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &w); err != nil {
		return telnyxInbound{}, err
	}
	in := telnyxInbound{
		EventType: w.Data.EventType,
		From:      w.Data.Payload.From.PhoneNumber,
		Text:      w.Data.Payload.Text,
	}
	if len(w.Data.Payload.ToList) > 0 {
		in.To = w.Data.Payload.ToList[0].PhoneNumber
	}
	return in, nil
}

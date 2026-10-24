// Package sms sends text messages. The provider is chosen by SMS_PROVIDER:
// log (write the text to the app log, send nothing) until the Telnyx
// toll-free number is verified, then telnyx.
package sms

import (
	"context"
	"regexp"
)

// Sender sends one text. providerID is the provider's message id ("" when
// it has none); status is what sms_message.status records.
type Sender interface {
	Send(ctx context.Context, toE164 string, body string) (providerID string, status string, err error)
}

// Logger is the part of the app logger the senders use
// (*logging.Logger satisfies it).
type Logger interface {
	Info(code string, message string, keyvals ...any)
}

// E164 is the phone number shape every stored and sent number has.
var E164 = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// MaskPhone keeps the last four digits, for logs.
func MaskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return "***" + phone[len(phone)-4:]
}

// LogSender writes each text to the log and reports it "logged".
type LogSender struct {
	Log Logger
}

func (s LogSender) Send(_ context.Context, toE164 string, body string) (string, string, error) {
	s.Log.Info("SMS_LOG", "text not sent (SMS_PROVIDER=log)", "to", MaskPhone(toE164), "body", body)
	return "", "logged", nil
}

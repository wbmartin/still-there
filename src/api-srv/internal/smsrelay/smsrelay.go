// Package smsrelay reads this app's queue on the SMS relay (src/sms-relay)
// when SMS_RELAY_URL, SMS_RELAY_TOKEN and SMS_RELAY_QUEUE are set: each
// poll receives up to 25 texts under a 60-second lease, hands each to the
// inbound handler and acks the ones it handled. A text whose handling
// failed is not acked and comes back when the lease ends. The relay
// delivers at least once; the handler skips a relay id it already
// recorded (watchdog.Inbound.HandleRelayed).
package smsrelay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"stillthere-api/internal/sms"
)

const (
	receiveLimit    = 25
	receiveLeaseSec = 60
	// clientTimeout keeps one call well inside the lease.
	clientTimeout    = 15 * time.Second
	maxResponseBytes = 1 << 20
)

// Message is one text as the relay returns it.
type Message struct {
	ID   int64  `json:"id"`
	From string `json:"from"`
	Text string `json:"text"`
}

// Client is the consumer half of the relay's API for one queue.
type Client struct {
	BaseURL string
	Token   string
	Queue   string
	// HTTP nil is a client with a 15-second timeout.
	HTTP *http.Client
}

func (c *Client) post(ctx context.Context, action string, body any, into any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(c.BaseURL, "/") + "/queue/" +
		url.PathEscape(c.Queue) + "/" + action
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: clientTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sms relay %s: %w", action, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("sms relay %s: %w", action, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sms relay %s: HTTP %d", action, resp.StatusCode)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("sms relay %s: reading response: %w", action, err)
	}
	return nil
}

// Receive leases up to limit texts for leaseSec seconds.
func (c *Client) Receive(ctx context.Context, limit int, leaseSec int) (string, []Message, error) {
	var out struct {
		LeaseID     string    `json:"leaseId"`
		MessageList []Message `json:"messageList"`
	}
	err := c.post(ctx, "receive",
		map[string]int{"limit": limit, "leaseSec": leaseSec}, &out)
	return out.LeaseID, out.MessageList, err
}

// Ack marks done the ids the lease holds; it returns how many the relay
// marked.
func (c *Client) Ack(ctx context.Context, leaseID string, idList []int64) (int, error) {
	var out struct {
		AckCount int `json:"ackCount"`
	}
	err := c.post(ctx, "ack",
		map[string]any{"leaseId": leaseID, "idList": idList}, &out)
	return out.AckCount, err
}

// Handler acts on one relayed text (*watchdog.Inbound).
type Handler interface {
	HandleRelayed(ctx context.Context, relayMessageID int64, from string, body string) error
}

// Logger is the part of the app logger the poller uses.
type Logger interface {
	Info(code string, message string, keyvals ...any)
	Error(code string, message string, keyvals ...any)
}

// Poller moves texts from the relay to the handler.
type Poller struct {
	Client  *Client
	Handler Handler
	Log     Logger
}

// Run polls at once and then every interval until ctx is done.
func (p *Poller) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		p.Poll(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Poll is one receive, handle, ack pass. A failure is logged and the next
// poll retries; a text with a number that is not E.164 is acked and
// dropped, as /smsInbound drops it.
func (p *Poller) Poll(ctx context.Context) {
	leaseID, messageList, err := p.Client.Receive(ctx, receiveLimit, receiveLeaseSec)
	if err != nil {
		p.Log.Error("SMS_RELAY_RECEIVE", "receiving texts", "err", err)
		return
	}
	var doneList []int64
	for _, m := range messageList {
		if !sms.E164.MatchString(m.From) {
			p.Log.Info("SMS_RELAY_DROP", "text without an E.164 sender", "relayId", m.ID)
			doneList = append(doneList, m.ID)
			continue
		}
		if err := p.Handler.HandleRelayed(ctx, m.ID, m.From, m.Text); err != nil {
			p.Log.Error("SMS_RELAY_HANDLE", "text not handled; it returns after its lease",
				"relayId", m.ID, "from", sms.MaskPhone(m.From), "err", err)
			continue
		}
		doneList = append(doneList, m.ID)
	}
	if len(doneList) == 0 {
		return
	}
	count, err := p.Client.Ack(ctx, leaseID, doneList)
	if err != nil {
		p.Log.Error("SMS_RELAY_ACK", "acking texts", "count", len(doneList), "err", err)
		return
	}
	if count != len(doneList) {
		// The lease ran out first: those texts come back, and their ids
		// are already recorded, so they are skipped then.
		p.Log.Error("SMS_RELAY_ACK", "relay acked fewer texts than handled",
			"handled", len(doneList), "acked", count)
	}
	p.Log.Info("SMS_RELAY_DONE", "texts handled from the relay", "count", count)
}

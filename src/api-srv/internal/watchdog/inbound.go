package watchdog

import (
	"context"
	"slices"
	"strings"

	"stillthere-api/internal/model"
	"stillthere-api/internal/repository"
	"stillthere-api/internal/sms"
)

// InboundStore is the slice of *repository.WatchdogRepository a reply
// needs.
type InboundStore interface {
	ContactListByPhone(ctx context.Context, phone string) ([]*repository.WatchContact, error)
	SetContactStatus(ctx context.Context, contactID int, securityContext int, status string, now float64) error
	AcknowledgeDeviceAlerts(ctx context.Context, securityContext int, deviceID int, actorName string, now float64) (int, error)
	InsertSMS(ctx context.Context, m repository.SMSRecord, now float64) error
	SMSRelayMessageSeen(ctx context.Context, relayMessageID int64) (bool, error)
}

// Inbound handles a text a person sent us: the keyword decides, the
// sender's number finds their contact rows (one per household they are
// on, possibly in more than one tenant).
type Inbound struct {
	Store  InboundStore
	Sender sms.Sender
	Log    Logger
	Now    func() float64
}

func (in *Inbound) now() float64 {
	if in.Now != nil {
		return in.Now()
	}
	return model.Now()
}

// Keywords, matched on the whole trimmed text, case-insensitively. STOP
// and its synonyms are the carriers' list; they get no reply from us, the
// carrier sends the confirmation.
var (
	stopWordList  = []string{"STOP", "STOPALL", "UNSUBSCRIBE", "CANCEL", "END", "QUIT"}
	startWordList = []string{"START", "UNSTOP"}
	helpWordList  = []string{"HELP", "INFO"}
	joinWordList  = []string{"Y", "YES"}
	ackWordList   = []string{"OK", "ACK"}
)

func keyword(body string) string {
	return strings.ToUpper(strings.Trim(strings.TrimSpace(body), ".!"))
}

func oneOf(word string, list []string) bool {
	for _, w := range list {
		if word == w {
			return true
		}
	}
	return false
}

// Handle acts on one text and returns the reply it sent ("" for none).
func (in *Inbound) Handle(ctx context.Context, from string, body string) (string, error) {
	return in.HandleInTenant(ctx, from, body, 0)
}

// HandleInTenant is Handle limited to one tenant's contacts (0: every
// tenant). The simulate route uses it, so an admin can stand in only for
// their own tenant's contacts.
func (in *Inbound) HandleInTenant(ctx context.Context, from string, body string, securityContext int) (string, error) {
	return in.handle(ctx, from, body, securityContext, nil)
}

// HandleRelayed is Handle for a text read from the SMS relay, which
// delivers at least once: an id already recorded on an inbound row was
// handled and is skipped. The id is recorded only once the text was acted
// on, so a text whose handling failed is handled again when it returns.
func (in *Inbound) HandleRelayed(ctx context.Context, relayMessageID int64, from string, body string) error {
	seen, err := in.Store.SMSRelayMessageSeen(ctx, relayMessageID)
	if err != nil || seen {
		return err
	}
	_, err = in.handle(ctx, from, body, 0, &relayMessageID)
	return err
}

// handle acts on the text, then records it (with its relay id, if any),
// then sends the reply. A failed step returns before the record.
func (in *Inbound) handle(ctx context.Context, from string, body string, securityContext int, relayMessageID *int64) (string, error) {
	now := in.now()
	contactList, err := in.Store.ContactListByPhone(ctx, from)
	if err != nil {
		return "", err
	}
	if securityContext != 0 {
		contactList = slices.DeleteFunc(contactList, func(c *repository.WatchContact) bool {
			return c.SecurityContextID != securityContext
		})
	}
	if len(contactList) == 0 {
		in.record(ctx, contactList, from, body, relayMessageID, now)
		in.Log.Info("SMS_INBOUND", "text from an unknown number", "from", sms.MaskPhone(from))
		return "", nil
	}

	word := keyword(body)
	reply := ""
	switch {
	case oneOf(word, stopWordList):
		for _, c := range contactList {
			if err := in.Store.SetContactStatus(ctx, c.ContactID, c.SecurityContextID, "opted_out", now); err != nil {
				return "", err
			}
		}
		in.record(ctx, contactList, from, body, relayMessageID, now)
		return "", nil
	case oneOf(word, joinWordList), oneOf(word, startWordList):
		// Y joins an invited contact; START also re-joins an opted-out one.
		start := oneOf(word, startWordList)
		var joined []string
		for _, c := range contactList {
			if c.ContactStatus != "invited" && !(start && c.ContactStatus == "opted_out") {
				continue
			}
			if err := in.Store.SetContactStatus(ctx, c.ContactID, c.SecurityContextID, "active", now); err != nil {
				return "", err
			}
			c.ContactStatus = "active"
			joined = append(joined, c.DeviceName)
		}
		if len(joined) == 0 {
			reply = UnknownReplyText()
		} else {
			reply = JoinedText(joined)
		}
	case oneOf(word, helpWordList):
		reply = HelpText()
	case oneOf(word, ackWordList):
		count := 0
		for _, c := range contactList {
			if c.ContactStatus != "active" {
				continue
			}
			n, err := in.Store.AcknowledgeDeviceAlerts(ctx, c.SecurityContextID, c.DeviceID, c.ContactName, now)
			if err != nil {
				return "", err
			}
			count += n
		}
		reply = AckReplyText(count)
	default:
		reply = UnknownReplyText()
	}

	in.record(ctx, contactList, from, body, relayMessageID, now)
	if target := textable(contactList); target != nil {
		providerID, status, err := in.Sender.Send(ctx, from, reply)
		if err != nil {
			status = "failed"
			in.Log.Error("SMS_INBOUND", "reply failed", "to", sms.MaskPhone(from), "err", err)
		}
		rec := smsRecord(target, "out", from, reply, status)
		if providerID != "" {
			rec.ProviderMessageID = &providerID
		}
		if err := in.Store.InsertSMS(ctx, rec, now); err != nil {
			in.Log.Error("SMS_INBOUND", "recording reply", "err", err)
		}
	}
	return reply, nil
}

// textable is a contact row of this number we may still text; nil when
// the number opted out everywhere.
func textable(contactList []*repository.WatchContact) *repository.WatchContact {
	for _, c := range contactList {
		if c.ContactStatus != "opted_out" {
			return c
		}
	}
	return nil
}

func (in *Inbound) record(ctx context.Context, contactList []*repository.WatchContact, phone, body string, relayMessageID *int64, now float64) {
	var c *repository.WatchContact
	if len(contactList) > 0 {
		c = contactList[0]
	}
	rec := smsRecord(c, "in", phone, body, "received")
	rec.RelayMessageID = relayMessageID
	if err := in.Store.InsertSMS(ctx, rec, now); err != nil {
		in.Log.Error("SMS_INBOUND", "recording text", "err", err)
	}
}

func smsRecord(c *repository.WatchContact, direction, phone, body, status string) repository.SMSRecord {
	rec := repository.SMSRecord{Direction: direction, PhoneE164: phone, Body: body, Status: status}
	if c != nil {
		id := c.ContactID
		rec.SecurityContextID = c.SecurityContextID
		rec.ContactID = &id
	}
	return rec
}

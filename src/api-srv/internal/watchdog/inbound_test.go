package watchdog

import (
	"context"
	"strings"
	"testing"

	"stillthere-api/internal/repository"
)

type fakeInboundStore struct {
	contactList []*repository.WatchContact
	acked       map[int]string // deviceID -> actor
	smsList     []repository.SMSRecord
}

func (f *fakeInboundStore) ContactListByPhone(_ context.Context, phone string) ([]*repository.WatchContact, error) {
	var out []*repository.WatchContact
	for _, c := range f.contactList {
		if c.PhoneE164 == phone {
			copy := *c
			out = append(out, &copy)
		}
	}
	return out, nil
}

func (f *fakeInboundStore) SetContactStatus(_ context.Context, contactID int, _ int, status string, _ float64) error {
	for _, c := range f.contactList {
		if c.ContactID == contactID {
			c.ContactStatus = status
		}
	}
	return nil
}

func (f *fakeInboundStore) AcknowledgeDeviceAlerts(_ context.Context, _ int, deviceID int, actor string, _ float64) (int, error) {
	f.acked[deviceID] = actor
	return 1, nil
}

func (f *fakeInboundStore) InsertSMS(_ context.Context, m repository.SMSRecord, _ float64) error {
	f.smsList = append(f.smsList, m)
	return nil
}

func inboundSetup(status string) (*fakeInboundStore, *fakeSender, *Inbound) {
	store := &fakeInboundStore{
		acked: map[int]string{},
		contactList: []*repository.WatchContact{
			{ContactID: 1, SecurityContextID: 2, DeviceID: 1, DeviceName: "Mom's house", ContactName: "Ann", PhoneE164: "+15550000001", ContactStatus: status},
		},
	}
	sender := &fakeSender{}
	return store, sender, &Inbound{Store: store, Sender: sender, Log: nopLog{}, Now: func() float64 { return t0 }}
}

func TestInboundYesJoinsInvited(t *testing.T) {
	store, sender, in := inboundSetup("invited")
	reply, err := in.Handle(context.Background(), "+15550000001", " yes ")
	if err != nil {
		t.Fatal(err)
	}
	if store.contactList[0].ContactStatus != "active" || !strings.Contains(reply, "alerts for Mom's house") {
		t.Fatalf("YES: status %q reply %q", store.contactList[0].ContactStatus, reply)
	}
	if len(sender.sent) != 1 || len(store.smsList) != 2 {
		t.Fatalf("want the text in and the reply out recorded: sent %v, recorded %d", sender.sent, len(store.smsList))
	}
}

func TestInboundStopOptsOutSilentlyAndOnlyStartRejoins(t *testing.T) {
	store, sender, in := inboundSetup("active")
	if reply, _ := in.Handle(context.Background(), "+15550000001", "STOP"); reply != "" || len(sender.sent) != 0 {
		t.Fatalf("STOP: want no reply from us, got %q %v", reply, sender.sent)
	}
	if store.contactList[0].ContactStatus != "opted_out" {
		t.Fatal("STOP did not opt out")
	}
	in.Handle(context.Background(), "+15550000001", "Y")
	if store.contactList[0].ContactStatus != "opted_out" {
		t.Fatal("Y re-joined an opted-out contact; only START may")
	}
	if len(sender.sent) != 0 {
		t.Fatalf("texted an opted-out number: %v", sender.sent)
	}
	in.Handle(context.Background(), "+15550000001", "start")
	if store.contactList[0].ContactStatus != "active" || len(sender.sent) != 1 {
		t.Fatalf("START: status %q, sent %v", store.contactList[0].ContactStatus, sender.sent)
	}
}

func TestInboundOkAcknowledgesForActiveOnly(t *testing.T) {
	store, _, in := inboundSetup("active")
	reply, _ := in.Handle(context.Background(), "+15550000001", "ok")
	if store.acked[1] != "Ann" || !strings.Contains(reply, "acknowledged 1") {
		t.Fatalf("OK: acked %v reply %q", store.acked, reply)
	}

	store, _, in = inboundSetup("invited")
	in.Handle(context.Background(), "+15550000001", "OK")
	if len(store.acked) != 0 {
		t.Fatal("an invited (not yet joined) contact acknowledged an alert")
	}
}

func TestInboundUnknownNumberGetsNoReply(t *testing.T) {
	store, sender, in := inboundSetup("active")
	reply, _ := in.Handle(context.Background(), "+15559999999", "OK")
	if reply != "" || len(sender.sent) != 0 || len(store.smsList) != 1 || store.smsList[0].ContactID != nil {
		t.Fatalf("unknown number: reply %q sent %v recorded %+v", reply, sender.sent, store.smsList)
	}
}
